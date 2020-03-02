package integrations

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/bugsnag/bugsnag-go"

	"github.com/toggl/pipes-api/pkg/toggl"
)

type Storage struct {
	db *sql.DB
}

func NewStorage(db *sql.DB) *Storage {
	svc := &Storage{
		db: db,
	}

	return svc
}

func (ps *Storage) IsDown() bool {
	if _, err := ps.db.Exec("SELECT 1"); err != nil {
		return true
	}
	return false
}

func (ps *Storage) LoadPipe(workspaceID int, serviceID, pipeID string) (*Pipe, error) {
	key := PipesKey(serviceID, pipeID)
	return ps.loadPipeWithKey(workspaceID, key)
}

func (ps *Storage) loadPipeWithKey(workspaceID int, key string) (*Pipe, error) {
	rows, err := ps.db.Query(singlePipesSQL, workspaceID, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, rows.Err()
	}
	var pipe Pipe
	if err := ps.load(rows, &pipe); err != nil {
		return nil, err
	}
	return &pipe, nil
}

func (ps *Storage) loadPipes(workspaceID int) (map[string]*Pipe, error) {
	pipes := make(map[string]*Pipe)
	rows, err := ps.db.Query(selectPipesSQL, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var pipe Pipe
		if err := ps.load(rows, &pipe); err != nil {
			return nil, err
		}
		pipes[pipe.Key] = &pipe
	}
	return pipes, nil
}

func (ps *Storage) GetPipesFromQueue() ([]*Pipe, error) {
	var pipes []*Pipe
	rows, err := ps.db.Query(selectPipesFromQueueSQL)
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
			pipe, err := ps.loadPipeWithKey(workspaceID, key)
			if err != nil {
				return nil, err
			}
			pipes = append(pipes, pipe)
		}
	}
	return pipes, nil
}

func (ps *Storage) SetQueuedPipeSynced(pipe *Pipe) error {
	_, err := ps.db.Exec(setQueuedPipeSyncedSQL, pipe.WorkspaceID, pipe.Key)
	return err
}

func (ps *Storage) Save(p *Pipe) error {
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

func (ps *Storage) load(rows *sql.Rows, p *Pipe) error {
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
	p.ServiceID = strings.Split(key, ":")[0]
	return nil
}

func (ps *Storage) loadLastSync(p *Pipe) {
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

func (ps *Storage) Destroy(p *Pipe, workspaceID int) error {
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

func (ps *Storage) DeletePipeConnections(workspaceID int, pipeConnectionKey, pipeStatusKey string) (err error) {
	tx, err := ps.db.Begin()
	if err != nil {
		return
	}
	_, err = tx.Exec(deletePipeConnectionsSQL, workspaceID, pipeConnectionKey)
	if err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			err = rollbackErr
		}
		return
	}
	_, err = tx.Exec(deletePipeStatusSQL, workspaceID, pipeStatusKey)
	if err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			err = rollbackErr
		}

	}
	return
}

func (ps *Storage) LoadPipeStatus(workspaceID int, serviceID, pipeID string) (*Status, error) {
	key := PipesKey(serviceID, pipeID)
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
	var pipeStatus Status
	if err = json.Unmarshal(b, &pipeStatus); err != nil {
		return nil, err
	}
	pipeStatus.WorkspaceID = workspaceID
	pipeStatus.ServiceID = serviceID
	pipeStatus.PipeID = pipeID
	return &pipeStatus, nil
}

func (ps *Storage) loadPipeStatuses(workspaceID int) (map[string]*Status, error) {
	pipeStatuses := make(map[string]*Status)
	rows, err := ps.db.Query(selectPipeStatusSQL, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var pipeStatus Status
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

func (ps *Storage) savePipeStatus(p *Status) error {
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

func (ps *Storage) QueueAutomaticPipes() error {
	_, err := ps.db.Exec(queueAutomaticPipesSQL)
	return err
}

func (ps *Storage) DeletePipeByWorkspaceIDServiceID(workspaceID int, serviceID string) error {
	_, err := ps.db.Exec(deletePipeSQL, workspaceID, serviceID+"%")
	return err
}

func (ps *Storage) QueuePipeAsFirst(pipe *Pipe) error {
	_, err := ps.db.Exec(queuePipeAsFirstSQL, pipe.WorkspaceID, pipe.Key)
	return err
}

func (ps *Storage) GetAccounts(s ExternalService) (*toggl.AccountsResponse, error) {
	var result []byte
	rows, err := ps.db.Query(`
		SELECT data FROM imports
		WHERE workspace_id = $1 AND Key = $2
		ORDER by created_at DESC
		LIMIT 1
	`, s.GetWorkspaceID(), s.KeyFor("accounts"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, rows.Err()
	}
	if err := rows.Scan(&result); err != nil {
		return nil, err
	}

	var accountsResponse toggl.AccountsResponse
	err = json.Unmarshal(result, &accountsResponse)
	if err != nil {
		return nil, err
	}
	return &accountsResponse, nil
}

func (ps *Storage) FetchAccounts(s ExternalService) error {
	var response toggl.AccountsResponse
	accounts, err := s.Accounts()
	response.Accounts = accounts
	if err != nil {
		response.Error = err.Error()
	}

	b, err := json.Marshal(response)
	if err != nil {
		bugsnag.Notify(err)
		return err
	}
	_, err = ps.db.Exec(`
    INSERT INTO imports(workspace_id, Key, data, created_at)
    VALUES($1, $2, $3, NOW())
  	`, s.GetWorkspaceID(), s.KeyFor("accounts"), b)
	if err != nil {
		bugsnag.Notify(err)
		return err
	}
	return nil
}

func (ps *Storage) ClearImportFor(s ExternalService, pipeID string) error {
	_, err := ps.db.Exec(`
	    DELETE FROM imports
	    WHERE workspace_id = $1 AND Key = $2
	`, s.GetWorkspaceID(), s.KeyFor(pipeID))
	return err
}

// ========================== get/saveObject ===================================
func (ps *Storage) getObject(s ExternalService, pipeID string) ([]byte, error) {
	var result []byte
	rows, err := ps.db.Query(`
		SELECT data FROM imports
		WHERE workspace_id = $1 AND Key = $2
		ORDER by created_at DESC
		LIMIT 1
	`, s.GetWorkspaceID(), s.KeyFor(pipeID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, rows.Err()
	}
	if err := rows.Scan(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func (ps *Storage) saveObject(workspaceID int, objKey string, obj interface{}) error {
	b, err := json.Marshal(obj)
	if err != nil {
		bugsnag.Notify(err)
		return err
	}

	_, err = ps.db.Exec(`
	  INSERT INTO imports(workspace_id, Key, data, created_at)
    VALUES($1, $2, $3, NOW())
	`, workspaceID, objKey, b)
	if err != nil {
		bugsnag.Notify(err)
		return err
	}
	return nil
}

// =============================================================================

const (
	selectPipesSQL = `SELECT workspace_id, Key, data
    FROM pipes WHERE workspace_id = $1
  `
	singlePipesSQL = `SELECT workspace_id, Key, data
    FROM pipes WHERE workspace_id = $1
    AND Key = $2 LIMIT 1
  `
	deletePipeSQL = `DELETE FROM pipes
    WHERE workspace_id = $1
    AND Key LIKE $2
  `
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
	deletePipeConnectionsSQL = `DELETE FROM connections
    WHERE workspace_id = $1
    AND Key = $2
  `
	selectPipesFromQueueSQL = `SELECT workspace_id, Key
	FROM get_queued_pipes()`

	queueAutomaticPipesSQL = `SELECT queue_automatic_pipes()`

	queuePipeAsFirstSQL = `SELECT queue_pipe_as_first($1, $2)`

	setQueuedPipeSyncedSQL = `UPDATE queued_pipes
	SET synced_at = now()
	WHERE workspace_id = $1
	AND Key = $2
	AND locked_at IS NOT NULL
	AND synced_at IS NULL`
)

const (
	selectPipeStatusSQL = `SELECT Key, data
    FROM pipes_status
    WHERE workspace_id = $1
  `
	singlePipeStatusSQL = `SELECT data
    FROM pipes_status
    WHERE workspace_id = $1
    AND Key = $2 LIMIT 1
  `
	deletePipeStatusSQL = `DELETE FROM pipes_status
		WHERE workspace_id = $1
		AND Key LIKE $2
  `
	lastSyncSQL = `SELECT (data->>'sync_date')::timestamp with time zone
    FROM pipes_status
    WHERE workspace_id = $1
    AND Key = $2
  `
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
)
