package storage

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
	deletePipeConnectionsSQL = `DELETE FROM connections WHERE workspace_id = $1 AND Key = $2`
	truncatePipesSQL         = `DELETE FROM pipes WHERE 1=1`

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

	selectAuthorizationSQL = `SELECT workspace_id, service, workspace_token, data
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
	deleteAuthorizationSQL   = `DELETE FROM authorizations WHERE workspace_id = $1 AND service = $2`
	truncateAuthorizationSQL = `TRUNCATE TABLE authorizations`

	selectConnectionSQL = `SELECT Key, data FROM connections WHERE workspace_id = $1 AND Key = $2 LIMIT 1`
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

func (ps *PostgresStorage) LoadPipe(workspaceID int, sid integration.ID, pid integration.PipeID) (*pipe.Pipe, error) {
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

func (ps *PostgresStorage) DeleteIDMappings(workspaceID int, pipeConnectionKey, pipeStatusKey string) (err error) {
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
	return tx.Commit()
}

func (ps *PostgresStorage) LoadPipeStatus(workspaceID int, sid integration.ID, pid integration.PipeID) (*pipe.Status, error) {
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

func (ps *PostgresStorage) DeletePipesByWorkspaceIDServiceID(workspaceID int, serviceID integration.ID) error {
	_, err := ps.db.Exec(deletePipeSQL, workspaceID, serviceID+"%")
	return err
}

func (ps *PostgresStorage) SaveAuthorization(a *pipe.Authorization) error {
	_, err := ps.db.Exec(insertAuthorizationSQL, a.WorkspaceID, a.ServiceID, a.WorkspaceToken, a.Data)
	if err != nil {
		return err
	}
	return nil
}

func (ps *PostgresStorage) LoadAuthorization(workspaceID int, externalServiceID integration.ID, a *pipe.Authorization) error {
	rows, err := ps.db.Query(selectAuthorizationSQL, workspaceID, externalServiceID)
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		return rows.Err()
	}
	err = rows.Scan(&a.WorkspaceID, &a.ServiceID, &a.WorkspaceToken, &a.Data)
	if err != nil {
		return err
	}
	return nil
}

func (ps *PostgresStorage) DeleteAuthorization(workspaceID int, externalServiceID integration.ID) error {
	_, err := ps.db.Exec(deleteAuthorizationSQL, workspaceID, externalServiceID)
	return err
}

// LoadWorkspaceAuthorizations loads map with authorizations status for each externalService.
// Map format: map[externalServiceID]isAuthorized
func (ps *PostgresStorage) LoadWorkspaceAuthorizations(workspaceID int) (map[integration.ID]bool, error) {
	authorizations := make(map[integration.ID]bool)
	rows, err := ps.db.Query(`SELECT service FROM authorizations WHERE workspace_id = $1`, workspaceID)
	if err != nil {
		return authorizations, err
	}
	defer rows.Close()
	for rows.Next() {
		var service integration.ID
		if err := rows.Scan(&service); err != nil {
			return authorizations, err
		}
		authorizations[service] = true
	}
	return authorizations, nil
}

func (ps *PostgresStorage) SaveIDMapping(c *pipe.IDMapping) error {
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

func (ps *PostgresStorage) LoadIDMapping(workspaceID int, key string) (*pipe.IDMapping, error) {
	return ps.loadIDMapping(workspaceID, key)
}

func (ps *PostgresStorage) LoadReversedIDMapping(workspaceID int, key string) (*pipe.ReversedIDMapping, error) {
	connection, err := ps.loadIDMapping(workspaceID, key)
	if err != nil {
		return nil, err
	}
	reversed := pipe.NewReversedConnection()
	for key, value := range connection.Data {
		reversed.Data[value] = key
	}
	return reversed, nil
}

func (ps *PostgresStorage) LoadPipes(workspaceID int) (map[string]*pipe.Pipe, error) {
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

func (ps *PostgresStorage) LoadLastSync(p *pipe.Pipe) {
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

func (ps *PostgresStorage) LoadPipeStatuses(workspaceID int) (map[string]*pipe.Status, error) {
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

func (ps *PostgresStorage) SavePipeStatus(p *pipe.Status) error {
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

func (ps *PostgresStorage) loadIDMapping(workspaceID int, key string) (*pipe.IDMapping, error) {
	rows, err := ps.db.Query(selectConnectionSQL, workspaceID, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	connection := pipe.NewIDMapping(workspaceID, key)
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
