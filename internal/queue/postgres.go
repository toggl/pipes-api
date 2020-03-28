package queue

import (
	"database/sql"

	"github.com/toggl/pipes-api/pkg/domain"
)

const (
	selectPipesFromQueueSQL = `SELECT workspace_id, Key FROM get_queued_pipes()`
	queueAutomaticPipesSQL  = `SELECT queue_automatic_pipes()`
	queuePipeAsFirstSQL     = `SELECT queue_pipe_as_first($1, $2)`
	setQueuedPipeSyncedSQL  = `UPDATE queued_pipes SET synced_at = now() WHERE workspace_id = $1 AND Key = $2 AND locked_at IS NOT NULL AND synced_at IS NULL`
)

type PostgresQueue struct {
	db *sql.DB
	*domain.PipeFactory
	domain.PipesStorage
}

func NewPostgresQueue(db *sql.DB, factory *domain.PipeFactory, store domain.PipesStorage) *PostgresQueue {
	return &PostgresQueue{
		db:           db,
		PipeFactory:  factory,
		PipesStorage: store,
	}
}

func (pq *PostgresQueue) QueueAutomaticPipes() error {
	_, err := pq.db.Exec(queueAutomaticPipesSQL)
	return err
}

func (pq *PostgresQueue) GetPipesFromQueue() ([]*domain.Pipe, error) {
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

func (pq *PostgresQueue) SetQueuedPipeSynced(pipe *domain.Pipe) error {
	_, err := pq.db.Exec(setQueuedPipeSyncedSQL, pipe.WorkspaceID, pipe.Key)
	return err
}

func (pq *PostgresQueue) QueuePipeAsFirst(pipe *domain.Pipe) error {
	_, err := pq.db.Exec(queuePipeAsFirstSQL, pipe.WorkspaceID, pipe.Key)
	return err
}
