package queue

import (
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/toggl/pipes-api/pkg/integrations"
	"github.com/toggl/pipes-api/pkg/pipe"
)

type PostgresQueue struct {
	db *sql.DB
}

func NewPostgresQueue(db *sql.DB) *PostgresQueue {
	return &PostgresQueue{
		db: db,
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
			pipe, err := pq.loadPipeWithKey(workspaceID, key)
			if err != nil {
				return nil, err
			}
			pipes = append(pipes, pipe)
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

func (pq *PostgresQueue) loadPipeWithKey(workspaceID int, key string) (*pipe.Pipe, error) {
	rows, err := pq.db.Query(singlePipesSQL, workspaceID, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, rows.Err()
	}
	var pipe pipe.Pipe
	if err := pq.load(rows, &pipe); err != nil {
		return nil, err
	}
	return &pipe, nil
}

func (pq *PostgresQueue) load(rows *sql.Rows, p *pipe.Pipe) error {
	var wid int
	var b []byte
	var key string
	if err := rows.Scan(&wid, &key, &b); err != nil {
		return err
	}
	err := json.Unmarshal(b, p)
	if err != nil {
		return err
	}
	p.Key = key
	p.WorkspaceID = wid
	p.ServiceID = integrations.ExternalServiceID(strings.Split(key, ":")[0])
	return nil
}
