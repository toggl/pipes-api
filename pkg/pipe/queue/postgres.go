package queue

import (
	"database/sql"

	"github.com/toggl/pipes-api/pkg/pipe"
)

const (
	selectPipesFromQueueSQL = `SELECT workspace_id, Key FROM get_queued_pipes()`
	queueAutomaticPipesSQL  = `SELECT queue_automatic_pipes()`
	queuePipeAsFirstSQL     = `SELECT queue_pipe_as_first($1, $2)`
	setQueuedPipeSyncedSQL  = `UPDATE queued_pipes SET synced_at = now() WHERE workspace_id = $1 AND Key = $2 AND locked_at IS NOT NULL AND synced_at IS NULL`
)

type PostgresQueue struct {
	db    *sql.DB
	store pipe.PipesStorage
}

func NewPostgresQueue(db *sql.DB, store pipe.PipesStorage) *PostgresQueue {
	return &PostgresQueue{
		db:    db,
		store: store,
	}
}

func (pq *PostgresQueue) QueueAutomaticPipes() error {
	_, err := pq.db.Exec(queueAutomaticPipesSQL)
	return err
}

func (pq *PostgresQueue) GetPipesFromQueue() ([]*pipe.Pipe, error) {
	var pipes []*pipe.Pipe
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
			sid, pid := pipe.GetSidPidFromKey(key)
			p, err := pq.store.Load(workspaceID, sid, pid)
			if err != nil {
				return nil, err
			}
			pipes = append(pipes, p)
		}
	}
	return pipes, nil
}

func (pq *PostgresQueue) SetQueuedPipeSynced(pipe *pipe.Pipe) error {
	_, err := pq.db.Exec(setQueuedPipeSyncedSQL, pipe.WorkspaceID, pipe.Key)
	return err
}

func (pq *PostgresQueue) QueuePipeAsFirst(pipe *pipe.Pipe) error {
	_, err := pq.db.Exec(queuePipeAsFirstSQL, pipe.WorkspaceID, pipe.Key)
	return err
}
