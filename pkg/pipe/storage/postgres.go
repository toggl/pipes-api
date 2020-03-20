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

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(db *sql.DB) *PostgresStorage {
	svc := &PostgresStorage{
		db: db,
	}

	return svc
}

func (ps *PostgresStorage) IsDown() bool {
	if _, err := ps.db.Exec("SELECT 1"); err != nil {
		return true
	}
	return false
}

func (ps *PostgresStorage) LoadPipe(workspaceID int, sid integrations.ExternalServiceID, pid integrations.PipeID) (*pipe.Pipe, error) {
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

func (ps *PostgresStorage) LoadPipeStatus(workspaceID int, sid integrations.ExternalServiceID, pid integrations.PipeID) (*pipe.Status, error) {
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

func (ps *PostgresStorage) DeletePipesByWorkspaceIDServiceID(workspaceID int, serviceID integrations.ExternalServiceID) error {
	_, err := ps.db.Exec(deletePipeSQL, workspaceID, serviceID+"%")
	return err
}

func (ps *PostgresStorage) DeleteAccountsFor(s integrations.ExternalService) error {
	_, err := ps.db.Exec(clearImportsSQL, s.GetWorkspaceID(), s.KeyFor(integrations.AccountsPipe))
	return err
}

func (ps *PostgresStorage) DeleteUsersFor(s integrations.ExternalService) error {
	_, err := ps.db.Exec(clearImportsSQL, s.GetWorkspaceID(), s.KeyFor(integrations.UsersPipe))
	return err
}

func (ps *PostgresStorage) SaveAuthorization(a *pipe.Authorization) error {
	_, err := ps.db.Exec(insertAuthorizationSQL, a.WorkspaceID, a.ServiceID, a.WorkspaceToken, a.Data)
	if err != nil {
		return err
	}
	return nil
}

func (ps *PostgresStorage) LoadAuthorization(workspaceID int, externalServiceID integrations.ExternalServiceID) (*pipe.Authorization, error) {
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

func (ps *PostgresStorage) DeleteAuthorization(workspaceID int, externalServiceID integrations.ExternalServiceID) error {
	_, err := ps.db.Exec(deleteAuthorizationSQL, workspaceID, externalServiceID)
	return err
}

// LoadWorkspaceAuthorizations loads map with authorizations status for each externalService.
// Map format: map[externalServiceID]isAuthorized
func (ps *PostgresStorage) LoadWorkspaceAuthorizations(workspaceID int) (map[integrations.ExternalServiceID]bool, error) {
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

func (ps *PostgresStorage) loadObject(s integrations.ExternalService, pid integrations.PipeID) ([]byte, error) {
	var result []byte
	rows, err := ps.db.Query(loadImportsSQL, s.GetWorkspaceID(), s.KeyFor(pid))
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

func (ps *PostgresStorage) SaveAccountsFor(s integrations.ExternalService, res toggl.AccountsResponse) error {
	b, err := json.Marshal(res)
	if err != nil {
		bugsnag.Notify(err)
		return err
	}
	return ps.saveObject(s, integrations.AccountsPipe, b)
}

func (ps *PostgresStorage) SaveUsersFor(s integrations.ExternalService, res toggl.UsersResponse) error {
	b, err := json.Marshal(res)
	if err != nil {
		bugsnag.Notify(err)
		return err
	}

	return ps.saveObject(s, integrations.UsersPipe, b)
}

func (ps *PostgresStorage) SaveClientsFor(s integrations.ExternalService, res toggl.ClientsResponse) error {
	b, err := json.Marshal(res)
	if err != nil {
		bugsnag.Notify(err)
		return err
	}

	return ps.saveObject(s, integrations.ClientsPipe, b)
}

func (ps *PostgresStorage) SaveProjectsFor(s integrations.ExternalService, res toggl.ProjectsResponse) error {
	b, err := json.Marshal(res)
	if err != nil {
		bugsnag.Notify(err)
		return err
	}

	return ps.saveObject(s, integrations.ProjectsPipe, b)
}

func (ps *PostgresStorage) SaveTasksFor(s integrations.ExternalService, res toggl.TasksResponse) error {
	b, err := json.Marshal(res)
	if err != nil {
		bugsnag.Notify(err)
		return err
	}

	return ps.saveObject(s, integrations.TasksPipe, b)
}

func (ps *PostgresStorage) SaveTodoListsFor(s integrations.ExternalService, res toggl.TasksResponse) error {
	b, err := json.Marshal(res)
	if err != nil {
		bugsnag.Notify(err)
		return err
	}

	return ps.saveObject(s, integrations.TodoListsPipe, b)
}

func (ps *PostgresStorage) LoadAccountsFor(s integrations.ExternalService) (*toggl.AccountsResponse, error) {
	b, err := ps.loadObject(s, integrations.AccountsPipe)
	if err != nil || b == nil {
		return nil, err
	}

	var accountsResponse toggl.AccountsResponse
	err = json.Unmarshal(b, &accountsResponse)
	if err != nil {
		return nil, err
	}
	return &accountsResponse, nil
}

func (ps *PostgresStorage) LoadUsersFor(s integrations.ExternalService) (*toggl.UsersResponse, error) {
	b, err := ps.loadObject(s, integrations.UsersPipe)
	if err != nil || b == nil {
		return nil, err
	}

	var usersResponse toggl.UsersResponse
	err = json.Unmarshal(b, &usersResponse)
	if err != nil {
		return nil, err
	}
	return &usersResponse, nil
}

func (ps *PostgresStorage) LoadClientsFor(s integrations.ExternalService) (*toggl.ClientsResponse, error) {
	b, err := ps.loadObject(s, integrations.ClientsPipe)
	if err != nil || b == nil {
		return nil, err
	}

	var clientsResponse toggl.ClientsResponse
	err = json.Unmarshal(b, &clientsResponse)
	if err != nil {
		return nil, err
	}
	return &clientsResponse, nil
}

func (ps *PostgresStorage) LoadProjectsFor(s integrations.ExternalService) (*toggl.ProjectsResponse, error) {
	b, err := ps.loadObject(s, integrations.ProjectsPipe)
	if err != nil || b == nil {
		return nil, err
	}

	var projectsResponse toggl.ProjectsResponse
	err = json.Unmarshal(b, &projectsResponse)
	if err != nil {
		return nil, err
	}

	return &projectsResponse, nil
}

func (ps *PostgresStorage) LoadTodoListsFor(s integrations.ExternalService) (*toggl.TasksResponse, error) {
	b, err := ps.loadObject(s, integrations.TodoListsPipe)
	if err != nil || b == nil {
		return nil, err
	}

	var tasksResponse toggl.TasksResponse
	err = json.Unmarshal(b, &tasksResponse)
	if err != nil {
		return nil, err
	}
	return &tasksResponse, nil
}

func (ps *PostgresStorage) LoadTasksFor(s integrations.ExternalService) (*toggl.TasksResponse, error) {
	b, err := ps.loadObject(s, integrations.TasksPipe)
	if err != nil || b == nil {
		return nil, err
	}

	var tasksResponse toggl.TasksResponse
	err = json.Unmarshal(b, &tasksResponse)
	if err != nil {
		return nil, err
	}
	return &tasksResponse, nil
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
	var pipe pipe.Pipe
	if err := ps.load(rows, &pipe); err != nil {
		return nil, err
	}
	return &pipe, nil
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
	p.ServiceID = integrations.ExternalServiceID(strings.Split(key, ":")[0])
	return nil
}

func (ps *PostgresStorage) saveObject(s integrations.ExternalService, pid integrations.PipeID, b []byte) error {
	_, err := ps.db.Exec(saveImportsSQL, s.GetWorkspaceID(), s.KeyFor(pid), b)
	if err != nil {
		bugsnag.Notify(err)
		return err
	}
	return nil
}
