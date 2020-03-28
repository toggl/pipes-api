package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"

	"github.com/toggl/pipes-api/pkg/domain"
	"github.com/toggl/pipes-api/pkg/integration"
)

// PipeStorage SQL queries
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

type PipeStorage struct {
	db *sql.DB
}

func NewPipeStorage(db *sql.DB) *PipeStorage {
	return &PipeStorage{db: db}
}

func (ps *PipeStorage) IsDown() bool {
	if _, err := ps.db.Exec("SELECT 1"); err != nil {
		return true
	}
	return false
}

func (ps *PipeStorage) Load(p *domain.Pipe) error {
	key := domain.PipesKey(p.ServiceID, p.ID)
	return ps.loadPipeWithKey(p.WorkspaceID, key, p)
}

func (ps *PipeStorage) Save(p *domain.Pipe) error {
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

func (ps *PipeStorage) Delete(p *domain.Pipe, workspaceID int) error {
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

func (ps *PipeStorage) LoadStatus(workspaceID int, sid integration.ID, pid integration.PipeID) (*domain.Status, error) {
	key := domain.PipesKey(sid, pid)
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
	var pipeStatus domain.Status
	if err = json.Unmarshal(b, &pipeStatus); err != nil {
		return nil, err
	}
	pipeStatus.WorkspaceID = workspaceID
	pipeStatus.ServiceID = sid
	pipeStatus.PipeID = pid
	return &pipeStatus, nil
}

func (ps *PipeStorage) DeleteByWorkspaceIDServiceID(workspaceID int, serviceID integration.ID) error {
	_, err := ps.db.Exec(deletePipeSQL, workspaceID, serviceID+"%")
	return err
}

func (ps *PipeStorage) LoadAll(workspaceID int) (map[string]*domain.Pipe, error) {
	pipes := make(map[string]*domain.Pipe)
	rows, err := ps.db.Query(selectPipesSQL, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var pipe domain.Pipe
		if err := ps.load(rows, &pipe); err != nil {
			return nil, err
		}
		pipes[pipe.Key] = &pipe
	}
	return pipes, nil
}

func (ps *PipeStorage) LoadLastSyncFor(p *domain.Pipe) {
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

func (ps *PipeStorage) LoadAllStatuses(workspaceID int) (map[string]*domain.Status, error) {
	pipeStatuses := make(map[string]*domain.Status)
	rows, err := ps.db.Query(selectPipeStatusSQL, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var pipeStatus domain.Status
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

func (ps *PipeStorage) SaveStatus(p *domain.Status) error {
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

func (ps *PipeStorage) loadPipeWithKey(workspaceID int, key string, p *domain.Pipe) error {
	rows, err := ps.db.Query(singlePipesSQL, workspaceID, key)
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		return rows.Err()
	}
	if err := ps.load(rows, p); err != nil {
		return err
	}
	return nil
}

func (ps *PipeStorage) load(rows *sql.Rows, p *domain.Pipe) error {
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
