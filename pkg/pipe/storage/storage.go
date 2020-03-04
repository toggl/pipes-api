package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"

	"github.com/bugsnag/bugsnag-go"

	"github.com/toggl/pipes-api/pkg/integrations"
	"github.com/toggl/pipes-api/pkg/pipe"
	"github.com/toggl/pipes-api/pkg/toggl"
)

const (
	selectPipesSQL = `SELECT workspace_id, Key, data
    FROM store WHERE workspace_id = $1
  `
	singlePipesSQL = `SELECT workspace_id, Key, data
    FROM store WHERE workspace_id = $1
    AND Key = $2 LIMIT 1
  `
	deletePipeSQL = `DELETE FROM store
    WHERE workspace_id = $1
    AND Key LIKE $2
  `
	insertPipesSQL = `
    WITH existing_pipe AS (
      UPDATE store SET data = $3
      WHERE workspace_id = $1 AND Key = $2
      RETURNING Key
    ),
    inserted_pipe AS (
      INSERT INTO store(workspace_id, Key, data)
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

const (
	selectAuthorizationSQL = `SELECT
		workspace_id, service, workspace_token, data
		FROM authorizations
		WHERE workspace_id = $1
		AND service = $2
		LIMIT 1
  `
	insertAuthorizationSQL = `WITH existing_auth AS (
		UPDATE authorizations SET data = $4, workspace_token = $3
		WHERE workspace_id = $1 AND service = $2
		RETURNING service
	),
	inserted_auth AS (
		INSERT INTO
		authorizations(workspace_id, service, workspace_token, data)
		SELECT $1, $2, $3, $4
		WHERE NOT EXISTS (SELECT 1 FROM existing_auth)
		RETURNING service
	)
	SELECT * FROM inserted_auth
	UNION
	SELECT * FROM existing_auth
  `
	deleteAuthorizationSQL = `DELETE FROM authorizations
		WHERE workspace_id = $1
		AND service = $2
	`
	truncateAuthorizationSQL = `TRUNCATE TABLE authorizations`
)

const (
	selectConnectionSQL = `SELECT Key, data
    FROM connections WHERE
    workspace_id = $1
    AND Key = $2
    LIMIT 1
  `
	insertConnectionSQL = `
    WITH existing_connection AS (
      UPDATE connections SET data = $3
      WHERE workspace_id = $1 AND Key = $2
      RETURNING Key
    ),
    inserted_connection AS (
      INSERT INTO connections(workspace_id, Key, data)
      SELECT $1, $2, $3
      WHERE NOT EXISTS (SELECT 1 FROM existing_connection)
      RETURNING Key
    )
    SELECT * FROM inserted_connection
    UNION
    SELECT * FROM existing_connection
  `
	truncateConnectionSQL = `TRUNCATE TABLE connections`
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

func (ps *Storage) LoadPipe(workspaceID int, sid integrations.ExternalServiceID, pid integrations.PipeID) (*pipe.Pipe, error) {
	key := pipe.PipesKey(sid, pid)
	return ps.loadPipeWithKey(workspaceID, key)
}

func (ps *Storage) GetPipesFromQueue() ([]*pipe.Pipe, error) {
	var pipes []*pipe.Pipe
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

func (ps *Storage) SetQueuedPipeSynced(pipe *pipe.Pipe) error {
	_, err := ps.db.Exec(setQueuedPipeSyncedSQL, pipe.WorkspaceID, pipe.Key)
	return err
}

func (ps *Storage) Save(p *pipe.Pipe) error {
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

func (ps *Storage) Destroy(p *pipe.Pipe, workspaceID int) error {
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

func (ps *Storage) LoadPipeStatus(workspaceID int, sid integrations.ExternalServiceID, pid integrations.PipeID) (*pipe.Status, error) {
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

func (ps *Storage) QueueAutomaticPipes() error {
	_, err := ps.db.Exec(queueAutomaticPipesSQL)
	return err
}

func (ps *Storage) DeletePipeByWorkspaceIDServiceID(workspaceID int, serviceID integrations.ExternalServiceID) error {
	_, err := ps.db.Exec(deletePipeSQL, workspaceID, serviceID+"%")
	return err
}

func (ps *Storage) QueuePipeAsFirst(pipe *pipe.Pipe) error {
	_, err := ps.db.Exec(queuePipeAsFirstSQL, pipe.WorkspaceID, pipe.Key)
	return err
}

func (ps *Storage) GetAccounts(s integrations.ExternalService) (*toggl.AccountsResponse, error) {
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

func (ps *Storage) FetchAccounts(s integrations.ExternalService) error {
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

func (ps *Storage) ClearImportFor(s integrations.ExternalService, pipeID integrations.PipeID) error {
	_, err := ps.db.Exec(`
	    DELETE FROM imports
	    WHERE workspace_id = $1 AND Key = $2
	`, s.GetWorkspaceID(), s.KeyFor(pipeID))
	return err
}

func (ps *Storage) SaveAuthorization(a *pipe.Authorization) error {
	_, err := ps.db.Exec(insertAuthorizationSQL, a.WorkspaceID, a.ServiceID, a.WorkspaceToken, a.Data)
	if err != nil {
		return err
	}
	return nil
}

func (ps *Storage) LoadAuthorization(workspaceID int, externalServiceID integrations.ExternalServiceID) (*pipe.Authorization, error) {
	rows, err := ps.db.Query(selectAuthorizationSQL, workspaceID, externalServiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, rows.Err()
	}
	var a pipe.Authorization
	err = rows.Scan(&a.WorkspaceID, &a.ServiceID, &a.WorkspaceToken, &a.Data)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (ps *Storage) DestroyAuthorization(workspaceID int, externalServiceID integrations.ExternalServiceID) error {
	_, err := ps.db.Exec(deleteAuthorizationSQL, workspaceID, externalServiceID)
	return err
}

// LoadWorkspaceAuthorizations loads map with authorizations status for each externalService.
// Map format: map[externalServiceID]isAuthorized
func (ps *Storage) LoadWorkspaceAuthorizations(workspaceID int) (map[integrations.ExternalServiceID]bool, error) {
	authorizations := make(map[integrations.ExternalServiceID]bool)
	rows, err := ps.db.Query(`SELECT service FROM authorizations WHERE workspace_id = $1`, workspaceID)
	if err != nil {
		return authorizations, err
	}
	defer rows.Close()
	for rows.Next() {
		var service integrations.ExternalServiceID
		if err := rows.Scan(&service); err != nil {
			return authorizations, err
		}
		authorizations[service] = true
	}
	return authorizations, nil
}

func (ps *Storage) SaveConnection(c *pipe.Connection) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	_, err = ps.db.Exec(insertConnectionSQL, c.WorkspaceID, c.Key, b)
	if err != nil {
		return err
	}
	return nil
}

func (ps *Storage) LoadConnection(workspaceID int, key string) (*pipe.Connection, error) {
	return ps.loadConnection(workspaceID, key)
}

func (ps *Storage) LoadReversedConnection(workspaceID int, key string) (*pipe.ReversedConnection, error) {
	connection, err := ps.loadConnection(workspaceID, key)
	if err != nil {
		return nil, err
	}
	reversed := pipe.NewReversedConnection()
	for key, value := range connection.Data {
		reversed.Data[value] = key
	}
	return reversed, nil
}

func (ps *Storage) loadConnection(workspaceID int, key string) (*pipe.Connection, error) {
	rows, err := ps.db.Query(selectConnectionSQL, workspaceID, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	connection := pipe.NewConnection(workspaceID, key)
	if rows.Next() {
		var b []byte
		if err := rows.Scan(&connection.Key, &b); err != nil {
			return nil, err
		}
		err := json.Unmarshal(b, connection)
		if err != nil {
			return nil, err
		}
	}
	return connection, nil
}

func (ps *Storage) loadPipeWithKey(workspaceID int, key string) (*pipe.Pipe, error) {
	rows, err := ps.db.Query(singlePipesSQL, workspaceID, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, rows.Err()
	}
	var pipe pipe.Pipe
	if err := ps.load(rows, &pipe); err != nil {
		return nil, err
	}
	return &pipe, nil
}

func (ps *Storage) LoadPipes(workspaceID int) (map[string]*pipe.Pipe, error) {
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

func (ps *Storage) load(rows *sql.Rows, p *pipe.Pipe) error {
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

func (ps *Storage) LoadLastSync(p *pipe.Pipe) {
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

func (ps *Storage) LoadPipeStatuses(workspaceID int) (map[string]*pipe.Status, error) {
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

func (ps *Storage) SavePipeStatus(p *pipe.Status) error {
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

func (ps *Storage) GetObject(s integrations.ExternalService, pid integrations.PipeID) ([]byte, error) {
	var result []byte
	rows, err := ps.db.Query(`
		SELECT data FROM imports
		WHERE workspace_id = $1 AND Key = $2
		ORDER by created_at DESC
		LIMIT 1
	`, s.GetWorkspaceID(), s.KeyFor(pid))
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

func (ps *Storage) SaveObject(workspaceID int, objKey string, obj interface{}) error {
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
