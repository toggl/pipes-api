package pipe

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"

	"github.com/toggl/pipes-api/pkg/integration"
	"github.com/toggl/pipes-api/pkg/pipe"
)

// PostgresStorage SQL queries
const (
	selectPipesSQL = `SELECT workspace_id, Key, data FROM pipes WHERE workspace_id = $1`
	singlePipesSQL = `SELECT workspace_id, Key, data FROM pipes WHERE workspace_id = $1 AND Key = $2 LIMIT 1`
	deletePipeSQL  = `DELETE FROM pipes WHERE workspace_id = $1 AND Key LIKE $2`
	insertPipesSQL = `
    WITH existing_pipe AS (
      UPDATE pipes SET data = $3
      WHERE workspace_id = $1 AND Key = $2
      RETURNING Key
    ),
    inserted_pipe AS (
      INSERT INTO pipes(workspace_id, Key, data)
      SELECT $1, $2, $3
      WHERE NOT EXISTS (SELECT 1 FROM existing_pipe)
      RETURNING Key
    )
    SELECT * FROM inserted_pipe
    UNION
    SELECT * FROM existing_pipe
  `
	truncatePipesSQL = `DELETE FROM pipes WHERE 1=1`

	selectPipeStatusSQL = `SELECT Key, data FROM pipes_status WHERE workspace_id = $1`
	singlePipeStatusSQL = `SELECT data FROM pipes_status WHERE workspace_id = $1 AND Key = $2 LIMIT 1`
	deletePipeStatusSQL = `DELETE FROM pipes_status WHERE workspace_id = $1 AND Key LIKE $2`
	lastSyncSQL         = `SELECT (data->>'sync_date')::timestamp with time zone FROM pipes_status WHERE workspace_id = $1 AND Key = $2`
	insertPipeStatusSQL = `
    WITH existing_status AS (
      UPDATE pipes_status SET data = $3
      WHERE workspace_id = $1 AND Key = $2
      RETURNING Key
    ),
    inserted_status AS (
      INSERT INTO pipes_status(workspace_id, Key, data)
      SELECT $1, $2, $3
      WHERE NOT EXISTS (SELECT 1 FROM existing_status)
      RETURNING Key
    )
    SELECT * FROM inserted_status
    UNION
    SELECT * FROM existing_status
  `
	truncatePipesStatusSQL = `TRUNCATE TABLE pipes_status`
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(db *sql.DB) *PostgresStorage {
	return &PostgresStorage{db: db}
}

func (ps *PostgresStorage) IsDown() bool {
	if _, err := ps.db.Exec("SELECT 1"); err != nil {
		return true
	}
	return false
}

func (ps *PostgresStorage) Load(workspaceID int, sid integration.ID, pid integration.PipeID) (*pipe.Pipe, error) {
	key := pipe.PipesKey(sid, pid)
	return ps.loadPipeWithKey(workspaceID, key)
}

func (ps *PostgresStorage) Save(p *pipe.Pipe) error {
	p.Configured = true
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = ps.db.Exec(insertPipesSQL, p.WorkspaceID, p.Key, b)
	if err != nil {
		return err
	}
	return nil
}

func (ps *PostgresStorage) Delete(p *pipe.Pipe, workspaceID int) error {
	tx, err := ps.db.Begin()
	if err != nil {
		return err
	}
	if _, err = tx.Exec(deletePipeSQL, workspaceID, p.Key); err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			return rollbackErr
		}
		return err
	}
	if _, err = tx.Exec(deletePipeStatusSQL, workspaceID, p.Key); err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			return rollbackErr
		}
		return err
	}
	return tx.Commit()
}

func (ps *PostgresStorage) LoadStatus(workspaceID int, sid integration.ID, pid integration.PipeID) (*pipe.Status, error) {
	key := pipe.PipesKey(sid, pid)
	rows, err := ps.db.Query(singlePipeStatusSQL, workspaceID, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, rows.Err()
	}
	var b []byte
	if err := rows.Scan(&b); err != nil {
		return nil, err
	}
	var pipeStatus pipe.Status
	if err = json.Unmarshal(b, &pipeStatus); err != nil {
		return nil, err
	}
	pipeStatus.WorkspaceID = workspaceID
	pipeStatus.ServiceID = sid
	pipeStatus.PipeID = pid
	return &pipeStatus, nil
}

func (ps *PostgresStorage) DeleteByWorkspaceIDServiceID(workspaceID int, serviceID integration.ID) error {
	_, err := ps.db.Exec(deletePipeSQL, workspaceID, serviceID+"%")
	return err
}

func (ps *PostgresStorage) LoadAll(workspaceID int) (map[string]*pipe.Pipe, error) {
	pipes := make(map[string]*pipe.Pipe)
	rows, err := ps.db.Query(selectPipesSQL, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var pipe pipe.Pipe
		if err := ps.load(rows, &pipe); err != nil {
			return nil, err
		}
		pipes[pipe.Key] = &pipe
	}
	return pipes, nil
}

func (ps *PostgresStorage) LoadLastSyncFor(p *pipe.Pipe) {
	err := ps.db.QueryRow(lastSyncSQL, p.WorkspaceID, p.Key).Scan(&p.LastSync)
	if err != nil {
		var err error
		t := time.Now()
		date := struct {
			StartDate string `json:"start_date"`
		}{}
		if err = json.Unmarshal(p.ServiceParams, &date); err == nil {
			t, _ = time.Parse("2006-01-02", date.StartDate)
		}
		p.LastSync = &t
	}
}

func (ps *PostgresStorage) LoadAllStatuses(workspaceID int) (map[string]*pipe.Status, error) {
	pipeStatuses := make(map[string]*pipe.Status)
	rows, err := ps.db.Query(selectPipeStatusSQL, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var pipeStatus pipe.Status
		var b []byte
		var key string
		if err := rows.Scan(&key, &b); err != nil {
			return nil, err
		}
		err := json.Unmarshal(b, &pipeStatus)
		if err != nil {
			return nil, err
		}
		pipeStatus.Key = key
		pipeStatuses[pipeStatus.Key] = &pipeStatus
	}
	return pipeStatuses, nil
}

func (ps *PostgresStorage) SaveStatus(p *pipe.Status) error {
	if p.Status == "success" {
		if len(p.ObjectCounts) > 0 {
			p.Message = fmt.Sprintf("%s successfully imported/exported", strings.Join(p.ObjectCounts, ", "))
		} else {
			p.Message = fmt.Sprintf("No new %s were imported/exported", p.PipeID)
		}
	}
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = ps.db.Exec(insertPipeStatusSQL, p.WorkspaceID, p.Key, b)
	if err != nil {
		return err
	}
	return nil
}

func (ps *PostgresStorage) loadPipeWithKey(workspaceID int, key string) (*pipe.Pipe, error) {
	rows, err := ps.db.Query(singlePipesSQL, workspaceID, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, rows.Err()
	}
	var p pipe.Pipe
	if err := ps.load(rows, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (ps *PostgresStorage) load(rows *sql.Rows, p *pipe.Pipe) error {
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
	p.ServiceID = integration.ID(strings.Split(key, ":")[0])
	return nil
}
