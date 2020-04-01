package integration

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"code.google.com/p/goauth2/oauth"
	"github.com/toggl/go-basecamp"

	"github.com/toggl/pipes-api/pkg/domain"
)

type BaseCampPipeIntegration struct {
	WorkspaceID int
	*BasecampParams
	token         oauth.Token
	modifiedSince *time.Time
}

type BasecampParams struct {
	AccountID int `json:"account_id"`
}

func (s *BaseCampPipeIntegration) ID() domain.IntegrationID {
	return domain.BaseCamp
}

func (s *BaseCampPipeIntegration) GetWorkspaceID() int {
	return s.WorkspaceID
}

func (s *BaseCampPipeIntegration) KeyFor(objectType domain.PipeID) string {
	if s.BasecampParams == nil {
		return fmt.Sprintf("basecamp:account:%s", objectType)
	}
	return fmt.Sprintf("basecamp:account:%d:%s", s.AccountID, objectType)
}

func (s *BaseCampPipeIntegration) SetParams(b []byte) error {
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s.BasecampParams == nil || s.AccountID == 0 {
		return errors.New("account_id must be present")
	}
	return nil
}

func (s *BaseCampPipeIntegration) SetAuthData(b []byte) error {
	return json.Unmarshal(b, &s.token)
}

func (s *BaseCampPipeIntegration) SetSince(since *time.Time) {
	s.modifiedSince = since
}

// Map basecamp accounts to local accounts
func (s *BaseCampPipeIntegration) Accounts() ([]*domain.Account, error) {
	foreignObjects, err := s.client().GetAccounts() // This will work only for Basecamp 2 account.
	if err != nil {
		return nil, err
	}
	var accounts []*domain.Account
	for _, object := range foreignObjects {
		account := domain.Account{
			ID:   int64(object.Id),
			Name: object.Name,
		}
		accounts = append(accounts, &account)
	}
	return accounts, nil
}

// Map basecamp people to local users
func (s *BaseCampPipeIntegration) Users() ([]*domain.User, error) {
	foreignObjects, err := s.client().GetPeople(s.AccountID)
	if err != nil {
		return nil, err
	}
	var users []*domain.User
	for _, object := range foreignObjects {
		user := domain.User{
			ForeignID: strconv.Itoa(object.Id),
			Name:      object.Name,
			Email:     object.Email,
		}
		users = append(users, &user)
	}
	return users, nil
}

// There are no clients in basecamp
func (s *BaseCampPipeIntegration) Clients() ([]*domain.Client, error) {
	return []*domain.Client{}, nil
}

// Map basecamp projects to projects
func (s *BaseCampPipeIntegration) Projects() ([]*domain.Project, error) {
	foreignObjects, err := s.client().GetProjects(s.AccountID)
	if err != nil {
		return nil, err
	}
	var projects []*domain.Project
	for _, object := range foreignObjects {
		if object.UpdatedAt.Before(*s.modifiedSince) {
			continue
		}
		project := domain.Project{
			Active:    true,
			ForeignID: strconv.Itoa(object.Id),
			Name:      object.Name,
		}
		projects = append(projects, &project)
	}
	return projects, nil
}

// Map basecamp todos to tasks
func (s *BaseCampPipeIntegration) Tasks() ([]*domain.Task, error) {
	c := s.client()
	foreignObjects, err := c.GetAllTodoLists(s.AccountID)
	if err != nil {
		return nil, err
	}
	var tasks []*domain.Task
	if len(foreignObjects) == 0 {
		return tasks, nil
	}
	for _, object := range foreignObjects {
		//if object.UpdatedAt.Before(*s.modifiedSince) {
		//	continue
		//}
		todoList, err := c.GetTodoList(s.AccountID, object.ProjectId, object.Id)
		if err != nil {
			return nil, err
		}
		if todoList == nil {
			continue
		}
		for _, todo := range todoList.Todos.Remaining {
			//if todo.UpdatedAt.Before(*s.modifiedSince) {
			// 	continue
			// }
			task := domain.Task{
				ForeignID:        strconv.Itoa(todo.Id),
				Name:             fmt.Sprintf("[%s] %s", object.Name, todo.Content),
				Active:           true,
				ForeignProjectID: strconv.Itoa(object.ProjectId),
			}
			tasks = append(tasks, &task)
		}
		for _, todo := range todoList.Todos.Completed {
			// if todo.UpdatedAt.Before(*s.modifiedSince) {
			// 	continue
			// }
			task := domain.Task{
				ForeignID:        strconv.Itoa(todo.Id),
				Name:             fmt.Sprintf("[%s] %s", object.Name, todo.Content),
				Active:           false,
				ForeignProjectID: strconv.Itoa(object.ProjectId),
			}
			tasks = append(tasks, &task)
		}
	}
	return tasks, nil
}

// Map basecamp todolists to tasks
func (s *BaseCampPipeIntegration) TodoLists() ([]*domain.Task, error) {
	foreignObjects, err := s.client().GetAllTodoLists(s.AccountID)
	if err != nil {
		return nil, err
	}
	var tasks []*domain.Task
	for _, object := range foreignObjects {
		if object.UpdatedAt.Before(*s.modifiedSince) {
			continue
		}
		task := domain.Task{
			ForeignID:        strconv.Itoa(object.Id),
			Name:             object.Name,
			Active:           !object.Completed,
			ForeignProjectID: strconv.Itoa(object.ProjectId),
		}
		tasks = append(tasks, &task)
	}
	return tasks, nil
}

func (s *BaseCampPipeIntegration) ExportTimeEntry(t *domain.TimeEntry) (int, error) {
	return 0, nil
}

func (s *BaseCampPipeIntegration) client() *basecamp.Client {
	return &basecamp.Client{
		ModifiedSince: s.modifiedSince,
		AccessToken:   s.token.AccessToken,
	}
}
