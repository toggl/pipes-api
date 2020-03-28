package storage

import (
	"database/sql"
	"encoding/json"

	_ "github.com/lib/pq"

	"github.com/bugsnag/bugsnag-go"

	"github.com/toggl/pipes-api/pkg/integration"
	"github.com/toggl/pipes-api/pkg/toggl"
)

const (
	loadImportsSQL = `
	SELECT data FROM imports
	WHERE workspace_id = $1 AND Key = $2
	ORDER by created_at DESC
	LIMIT 1
	`
	saveImportsSQL = `
	INSERT INTO imports(workspace_id, Key, data, created_at)
    VALUES($1, $2, $3, NOW())
	`

	clearImportsSQL = `
	    DELETE FROM imports
	    WHERE workspace_id = $1 AND Key = $2
	`

	truncateImportsSQL = `TRUNCATE TABLE imports`
)

type ImportStorage struct {
	db *sql.DB
}

func NewImportStorage(db *sql.DB) *ImportStorage {
	return &ImportStorage{db: db}
}

func (is *ImportStorage) SaveAccountsFor(s integration.Integration, res toggl.AccountsResponse) error {
	b, err := json.Marshal(res)
	if err != nil {
		return err
	}
	return is.saveObject(s, integration.AccountsPipe, b)
}

func (is *ImportStorage) SaveUsersFor(s integration.Integration, res toggl.UsersResponse) error {
	b, err := json.Marshal(res)
	if err != nil {
		return err
	}

	return is.saveObject(s, integration.UsersPipe, b)
}

func (is *ImportStorage) SaveClientsFor(s integration.Integration, res toggl.ClientsResponse) error {
	b, err := json.Marshal(res)
	if err != nil {
		return err
	}

	return is.saveObject(s, integration.ClientsPipe, b)
}

func (is *ImportStorage) SaveProjectsFor(s integration.Integration, res toggl.ProjectsResponse) error {
	b, err := json.Marshal(res)
	if err != nil {
		return err
	}

	return is.saveObject(s, integration.ProjectsPipe, b)
}

func (is *ImportStorage) SaveTasksFor(s integration.Integration, res toggl.TasksResponse) error {
	b, err := json.Marshal(res)
	if err != nil {
		return err
	}

	return is.saveObject(s, integration.TasksPipe, b)
}

func (is *ImportStorage) SaveTodoListsFor(s integration.Integration, res toggl.TasksResponse) error {
	b, err := json.Marshal(res)
	if err != nil {
		return err
	}

	return is.saveObject(s, integration.TodoListsPipe, b)
}

func (is *ImportStorage) LoadAccountsFor(s integration.Integration) (*toggl.AccountsResponse, error) {
	b, err := is.loadObject(s, integration.AccountsPipe)
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

func (is *ImportStorage) LoadUsersFor(s integration.Integration) (*toggl.UsersResponse, error) {
	b, err := is.loadObject(s, integration.UsersPipe)
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

func (is *ImportStorage) LoadClientsFor(s integration.Integration) (*toggl.ClientsResponse, error) {
	b, err := is.loadObject(s, integration.ClientsPipe)
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

func (is *ImportStorage) LoadProjectsFor(s integration.Integration) (*toggl.ProjectsResponse, error) {
	b, err := is.loadObject(s, integration.ProjectsPipe)
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

func (is *ImportStorage) LoadTodoListsFor(s integration.Integration) (*toggl.TasksResponse, error) {
	b, err := is.loadObject(s, integration.TodoListsPipe)
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

func (is *ImportStorage) LoadTasksFor(s integration.Integration) (*toggl.TasksResponse, error) {
	b, err := is.loadObject(s, integration.TasksPipe)
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

func (is *ImportStorage) DeleteAccountsFor(s integration.Integration) error {
	_, err := is.db.Exec(clearImportsSQL, s.GetWorkspaceID(), s.KeyFor(integration.AccountsPipe))
	return err
}

func (is *ImportStorage) DeleteUsersFor(s integration.Integration) error {
	_, err := is.db.Exec(clearImportsSQL, s.GetWorkspaceID(), s.KeyFor(integration.UsersPipe))
	return err
}

func (is *ImportStorage) loadObject(s integration.Integration, pid integration.PipeID) ([]byte, error) {
	var result []byte
	rows, err := is.db.Query(loadImportsSQL, s.GetWorkspaceID(), s.KeyFor(pid))
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

func (is *ImportStorage) saveObject(s integration.Integration, pid integration.PipeID, b []byte) error {
	_, err := is.db.Exec(saveImportsSQL, s.GetWorkspaceID(), s.KeyFor(pid), b)
	if err != nil {
		bugsnag.Notify(err)
		return err
	}
	return nil
}
