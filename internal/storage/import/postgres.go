package _import

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

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(db *sql.DB) *PostgresStorage {
	return &PostgresStorage{db: db}
}

func (pis *PostgresStorage) SaveAccountsFor(s integration.Integration, res toggl.AccountsResponse) error {
	b, err := json.Marshal(res)
	if err != nil {
		return err
	}
	return pis.saveObject(s, integration.AccountsPipe, b)
}

func (pis *PostgresStorage) SaveUsersFor(s integration.Integration, res toggl.UsersResponse) error {
	b, err := json.Marshal(res)
	if err != nil {
		return err
	}

	return pis.saveObject(s, integration.UsersPipe, b)
}

func (pis *PostgresStorage) SaveClientsFor(s integration.Integration, res toggl.ClientsResponse) error {
	b, err := json.Marshal(res)
	if err != nil {
		return err
	}

	return pis.saveObject(s, integration.ClientsPipe, b)
}

func (pis *PostgresStorage) SaveProjectsFor(s integration.Integration, res toggl.ProjectsResponse) error {
	b, err := json.Marshal(res)
	if err != nil {
		return err
	}

	return pis.saveObject(s, integration.ProjectsPipe, b)
}

func (pis *PostgresStorage) SaveTasksFor(s integration.Integration, res toggl.TasksResponse) error {
	b, err := json.Marshal(res)
	if err != nil {
		return err
	}

	return pis.saveObject(s, integration.TasksPipe, b)
}

func (pis *PostgresStorage) SaveTodoListsFor(s integration.Integration, res toggl.TasksResponse) error {
	b, err := json.Marshal(res)
	if err != nil {
		return err
	}

	return pis.saveObject(s, integration.TodoListsPipe, b)
}

func (pis *PostgresStorage) LoadAccountsFor(s integration.Integration) (*toggl.AccountsResponse, error) {
	b, err := pis.loadObject(s, integration.AccountsPipe)
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

func (pis *PostgresStorage) LoadUsersFor(s integration.Integration) (*toggl.UsersResponse, error) {
	b, err := pis.loadObject(s, integration.UsersPipe)
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

func (pis *PostgresStorage) LoadClientsFor(s integration.Integration) (*toggl.ClientsResponse, error) {
	b, err := pis.loadObject(s, integration.ClientsPipe)
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

func (pis *PostgresStorage) LoadProjectsFor(s integration.Integration) (*toggl.ProjectsResponse, error) {
	b, err := pis.loadObject(s, integration.ProjectsPipe)
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

func (pis *PostgresStorage) LoadTodoListsFor(s integration.Integration) (*toggl.TasksResponse, error) {
	b, err := pis.loadObject(s, integration.TodoListsPipe)
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

func (pis *PostgresStorage) LoadTasksFor(s integration.Integration) (*toggl.TasksResponse, error) {
	b, err := pis.loadObject(s, integration.TasksPipe)
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

func (pis *PostgresStorage) DeleteAccountsFor(s integration.Integration) error {
	_, err := pis.db.Exec(clearImportsSQL, s.GetWorkspaceID(), s.KeyFor(integration.AccountsPipe))
	return err
}

func (pis *PostgresStorage) DeleteUsersFor(s integration.Integration) error {
	_, err := pis.db.Exec(clearImportsSQL, s.GetWorkspaceID(), s.KeyFor(integration.UsersPipe))
	return err
}

func (pis *PostgresStorage) loadObject(s integration.Integration, pid integration.PipeID) ([]byte, error) {
	var result []byte
	rows, err := pis.db.Query(loadImportsSQL, s.GetWorkspaceID(), s.KeyFor(pid))
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

func (pis *PostgresStorage) saveObject(s integration.Integration, pid integration.PipeID, b []byte) error {
	_, err := pis.db.Exec(saveImportsSQL, s.GetWorkspaceID(), s.KeyFor(pid), b)
	if err != nil {
		bugsnag.Notify(err)
		return err
	}
	return nil
}
