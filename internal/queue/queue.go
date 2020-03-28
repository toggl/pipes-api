package queue

import (
	"database/sql"
	"errors"
	"sync"
	"time"

	"github.com/toggl/pipes-api/pkg/domain"
	"github.com/toggl/pipes-api/pkg/integration"
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
	db *sql.DB
	*domain.PipeFactory
	domain.PipesStorage
}

func NewPipesQueue(db *sql.DB, factory *domain.PipeFactory, store domain.PipesStorage) *Queue {
	return &Queue{
		db:           db,
		PipeFactory:  factory,
		PipesStorage: store,
	}
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

			p := pq.PipeFactory.Create(workspaceID, sid, pid)
			if err := pq.PipesStorage.Load(p); err != nil {
				return nil, err
			}
			pipes = append(pipes, p)
		}
	}
	return pipes, nil
}

func (pq *Queue) MarkPipeSynchronized(pipe *domain.Pipe) error {
	_, err := pq.db.Exec(setQueuedPipeSyncedSQL, pipe.WorkspaceID, pipe.Key)
	return err
}

func (pq *Queue) SchedulePipeSynchronization(pipe *domain.Pipe) error {
	// make sure no race condition on fetching workspace lock
	postPipeRunLock.Lock()
	wsLock, exists := postPipeRunWorkspaceLock[pipe.WorkspaceID]
	if !exists {
		wsLock = &sync.Mutex{}
		postPipeRunWorkspaceLock[pipe.WorkspaceID] = wsLock
	}
	postPipeRunLock.Unlock()

	if pipe.ID == integration.UsersPipe {
		if pipe.UsersSelector == nil {
			return domain.SetParamsError{errors.New("Missing request payload")}
		}

		go func() {
			wsLock.Lock()
			pipe.Synchronize()
			wsLock.Unlock()
		}()
		time.Sleep(500 * time.Millisecond) // TODO: Is that synchronization ? :D
		return nil
	}

	_, err := pq.db.Exec(queuePipeAsFirstSQL, pipe.WorkspaceID, pipe.Key)
	return err
}
