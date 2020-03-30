package storage

import (
	"database/sql"
	"encoding/json"

	_ "github.com/lib/pq"

	"github.com/bugsnag/bugsnag-go"

	"github.com/toggl/pipes-api/pkg/domain"
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
	DB *sql.DB
}

func (is *ImportStorage) SaveAccountsFor(s domain.PipeIntegration, res domain.AccountsResponse) error {
	b, err := json.Marshal(res)
	if err != nil {
		return err
	}
	return is.saveObject(s, domain.AccountsPipe, b)
}

func (is *ImportStorage) SaveUsersFor(s domain.PipeIntegration, res domain.UsersResponse) error {
	b, err := json.Marshal(res)
	if err != nil {
		return err
	}

	return is.saveObject(s, domain.UsersPipe, b)
}

func (is *ImportStorage) SaveClientsFor(s domain.PipeIntegration, res domain.ClientsResponse) error {
	b, err := json.Marshal(res)
	if err != nil {
		return err
	}

	return is.saveObject(s, domain.ClientsPipe, b)
}

func (is *ImportStorage) SaveProjectsFor(s domain.PipeIntegration, res domain.ProjectsResponse) error {
	b, err := json.Marshal(res)
	if err != nil {
		return err
	}

	return is.saveObject(s, domain.ProjectsPipe, b)
}

func (is *ImportStorage) SaveTasksFor(s domain.PipeIntegration, res domain.TasksResponse) error {
	b, err := json.Marshal(res)
	if err != nil {
		return err
	}

	return is.saveObject(s, domain.TasksPipe, b)
}

func (is *ImportStorage) SaveTodoListsFor(s domain.PipeIntegration, res domain.TasksResponse) error {
	b, err := json.Marshal(res)
	if err != nil {
		return err
	}

	return is.saveObject(s, domain.TodoListsPipe, b)
}

func (is *ImportStorage) LoadAccountsFor(s domain.PipeIntegration) (*domain.AccountsResponse, error) {
	b, err := is.loadObject(s, domain.AccountsPipe)
	if err != nil || b == nil {
		return nil, err
	}

	var accountsResponse domain.AccountsResponse
	err = json.Unmarshal(b, &accountsResponse)
	if err != nil {
		return nil, err
	}
	return &accountsResponse, nil
}

func (is *ImportStorage) LoadUsersFor(s domain.PipeIntegration) (*domain.UsersResponse, error) {
	b, err := is.loadObject(s, domain.UsersPipe)
	if err != nil || b == nil {
		return nil, err
	}

	var usersResponse domain.UsersResponse
	err = json.Unmarshal(b, &usersResponse)
	if err != nil {
		return nil, err
	}
	return &usersResponse, nil
}

func (is *ImportStorage) LoadClientsFor(s domain.PipeIntegration) (*domain.ClientsResponse, error) {
	b, err := is.loadObject(s, domain.ClientsPipe)
	if err != nil || b == nil {
		return nil, err
	}

	var clientsResponse domain.ClientsResponse
	err = json.Unmarshal(b, &clientsResponse)
	if err != nil {
		return nil, err
	}
	return &clientsResponse, nil
}

func (is *ImportStorage) LoadProjectsFor(s domain.PipeIntegration) (*domain.ProjectsResponse, error) {
	b, err := is.loadObject(s, domain.ProjectsPipe)
	if err != nil || b == nil {
		return nil, err
	}

	var projectsResponse domain.ProjectsResponse
	err = json.Unmarshal(b, &projectsResponse)
	if err != nil {
		return nil, err
	}

	return &projectsResponse, nil
}

func (is *ImportStorage) LoadTodoListsFor(s domain.PipeIntegration) (*domain.TasksResponse, error) {
	b, err := is.loadObject(s, domain.TodoListsPipe)
	if err != nil || b == nil {
		return nil, err
	}

	var tasksResponse domain.TasksResponse
	err = json.Unmarshal(b, &tasksResponse)
	if err != nil {
		return nil, err
	}
	return &tasksResponse, nil
}

func (is *ImportStorage) LoadTasksFor(s domain.PipeIntegration) (*domain.TasksResponse, error) {
	b, err := is.loadObject(s, domain.TasksPipe)
	if err != nil || b == nil {
		return nil, err
	}

	var tasksResponse domain.TasksResponse
	err = json.Unmarshal(b, &tasksResponse)
	if err != nil {
		return nil, err
	}
	return &tasksResponse, nil
}

func (is *ImportStorage) DeleteAccountsFor(s domain.PipeIntegration) error {
	_, err := is.DB.Exec(clearImportsSQL, s.GetWorkspaceID(), s.KeyFor(domain.AccountsPipe))
	return err
}

func (is *ImportStorage) DeleteUsersFor(s domain.PipeIntegration) error {
	_, err := is.DB.Exec(clearImportsSQL, s.GetWorkspaceID(), s.KeyFor(domain.UsersPipe))
	return err
}

func (is *ImportStorage) loadObject(s domain.PipeIntegration, pid domain.PipeID) ([]byte, error) {
	var result []byte
	rows, err := is.DB.Query(loadImportsSQL, s.GetWorkspaceID(), s.KeyFor(pid))
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

func (is *ImportStorage) saveObject(s domain.PipeIntegration, pid domain.PipeID, b []byte) error {
	_, err := is.DB.Exec(saveImportsSQL, s.GetWorkspaceID(), s.KeyFor(pid), b)
	if err != nil {
		bugsnag.Notify(err)
		return err
	}
	return nil
}
