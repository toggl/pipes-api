package sync

import (
	"database/sql"
	"errors"
	"sync"
	"time"

	"github.com/toggl/pipes-api/internal/service"
	"github.com/toggl/pipes-api/pkg/domain"
)

const (
	selectPipesFromQueueSQL = `SELECT workspace_id, Key FROM get_queued_pipes()`
	queueAutomaticPipesSQL  = `SELECT queue_automatic_pipes()`
	queuePipeAsFirstSQL     = `SELECT queue_pipe_as_first($1, $2)`
	setQueuedPipeSyncedSQL  = `UPDATE queued_pipes SET synced_at = now() WHERE workspace_id = $1 AND Key = $2 AND locked_at IS NOT NULL AND synced_at IS NULL`
)

// mutex to prevent multiple of postPipeRun on same workspace run at same time
var postPipeRunWorkspaceLock = map[int]*sync.Mutex{}
var postPipeRunLock sync.Mutex

type Queue struct {
	db              *sql.DB
	pipeSyncService domain.PipeSyncService
	pipeStorage     domain.PipesStorage
}

func NewQueue(db *sql.DB, pipeSyncService domain.PipeSyncService, pipesStorage domain.PipesStorage) *Queue {
	if db == nil {
		panic("Queue.db should not be nil")
	}
	if pipeSyncService == nil {
		panic("Queue.pipeSyncService should not be nil")
	}
	if pipesStorage == nil {
		panic("Queue.pipeStorage should not be nil")
	}
	return &Queue{db: db, pipeSyncService: pipeSyncService, pipeStorage: pipesStorage}
}

func (pq *Queue) ScheduleAutomaticPipesSynchronization() error {
	_, err := pq.db.Exec(queueAutomaticPipesSQL)
	return err
}

func (pq *Queue) LoadScheduledPipes() ([]*domain.Pipe, error) {
	var pipes []*domain.Pipe
	rows, err := pq.db.Query(selectPipesFromQueueSQL)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var workspaceID int
		var key string
		err := rows.Scan(&workspaceID, &key)
		if err != nil {
			return nil, err
		}

		if workspaceID > 0 && len(key) > 0 {
			sid, pid := domain.GetSidPidFromKey(key)

			p := domain.NewPipe(workspaceID, sid, pid)
			if err := pq.pipeStorage.Load(p); err != nil {
				return nil, err
			}
			pipes = append(pipes, p)
		}
	}
	return pipes, nil
}

func (pq *Queue) MarkPipeSynchronized(pipe *domain.Pipe) error {
	_, err := pq.db.Exec(setQueuedPipeSyncedSQL, pipe.WorkspaceID, pipe.Key())
	return err
}

func (pq *Queue) SchedulePipeSynchronization(workspaceID int, serviceID domain.IntegrationID, pipeID domain.PipeID, usersSelector domain.UserParams) error {
	p := domain.NewPipe(workspaceID, serviceID, pipeID)
	if err := pq.pipeStorage.Load(p); err != nil {
		return err
	}
	if p == nil {
		return service.ErrPipeNotConfigured
	}

	p.UsersSelector = usersSelector

	// make sure no race condition on fetching workspace lock
	postPipeRunLock.Lock()
	wsLock, exists := postPipeRunWorkspaceLock[p.WorkspaceID]
	if !exists {
		wsLock = &sync.Mutex{}
		postPipeRunWorkspaceLock[p.WorkspaceID] = wsLock
	}
	postPipeRunLock.Unlock()

	if p.ID == domain.UsersPipe {
		if len(p.UsersSelector.IDs) == 0 {
			return service.SetParamsError{errors.New("Missing request payload")}
		}

		go func() {
			wsLock.Lock()
			pq.pipeSyncService.Synchronize(p)
			wsLock.Unlock()
		}()
		time.Sleep(500 * time.Millisecond) // TODO: Is that synchronization ? :D
		return nil
	}

	return pq.queuePipeAsFirst(p.WorkspaceID, p.Key())
}

func (pq *Queue) queuePipeAsFirst(workspaceId int, key string) error {
	_, err := pq.db.Exec(queuePipeAsFirstSQL, workspaceId, key)
	return err
}
