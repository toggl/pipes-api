package basecamp

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"code.google.com/p/goauth2/oauth"
	"github.com/toggl/go-basecamp"

	"github.com/toggl/pipes-api/pkg/integration"
	"github.com/toggl/pipes-api/pkg/toggl"
)

type Service struct {
	WorkspaceID int
	*BasecampParams
	token         oauth.Token
	modifiedSince *time.Time
}

type BasecampParams struct {
	AccountID int `json:"account_id"`
}

func (s *Service) ID() integration.ID {
	return integration.BaseCamp
}

func (s *Service) GetWorkspaceID() int {
	return s.WorkspaceID
}

func (s *Service) KeyFor(objectType integration.PipeID) string {
	if s.BasecampParams == nil {
		return fmt.Sprintf("basecamp:account:%s", objectType)
	}
	return fmt.Sprintf("basecamp:account:%d:%s", s.AccountID, objectType)
}

func (s *Service) SetParams(b []byte) error {
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s.BasecampParams == nil || s.AccountID == 0 {
		return errors.New("account_id must be present")
	}
	return nil
}

func (s *Service) SetAuthData(b []byte) error {
	return json.Unmarshal(b, &s.token)
}

func (s *Service) SetSince(since *time.Time) {
	s.modifiedSince = since
}

// Map basecamp accounts to local accounts
func (s *Service) Accounts() ([]*toggl.Account, error) {
	foreignObjects, err := s.client().GetAccounts()
	if err != nil {
		return nil, err
	}
	var accounts []*toggl.Account
	for _, object := range foreignObjects {
		account := toggl.Account{
			ID:   int64(object.Id),
			Name: object.Name,
		}
		accounts = append(accounts, &account)
	}
	return accounts, nil
}

// Map basecamp people to local users
func (s *Service) Users() ([]*toggl.User, error) {
	foreignObjects, err := s.client().GetPeople(s.AccountID)
	if err != nil {
		return nil, err
	}
	var users []*toggl.User
	for _, object := range foreignObjects {
		user := toggl.User{
			ForeignID: strconv.Itoa(object.Id),
			Name:      object.Name,
			Email:     object.Email,
		}
		users = append(users, &user)
	}
	return users, nil
}

// There are no clients in basecamp
func (s *Service) Clients() ([]*toggl.Client, error) {
	return nil, nil
}

// Map basecamp projects to projects
func (s *Service) Projects() ([]*toggl.Project, error) {
	foreignObjects, err := s.client().GetProjects(s.AccountID)
	if err != nil {
		return nil, err
	}
	var projects []*toggl.Project
	for _, object := range foreignObjects {
		if object.UpdatedAt.Before(*s.modifiedSince) {
			continue
		}
		project := toggl.Project{
			Active:    true,
			ForeignID: strconv.Itoa(object.Id),
			Name:      object.Name,
		}
		projects = append(projects, &project)
	}
	return projects, nil
}

// Map basecamp todos to tasks
func (s *Service) Tasks() ([]*toggl.Task, error) {
	c := s.client()
	foreignObjects, err := c.GetAllTodoLists(s.AccountID)
	if err != nil {
		return nil, err
	}
	var tasks []*toggl.Task
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
			task := toggl.Task{
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
			task := toggl.Task{
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
func (s *Service) TodoLists() ([]*toggl.Task, error) {
	foreignObjects, err := s.client().GetAllTodoLists(s.AccountID)
	if err != nil {
		return nil, err
	}
	var tasks []*toggl.Task
	for _, object := range foreignObjects {
		if object.UpdatedAt.Before(*s.modifiedSince) {
			continue
		}
		task := toggl.Task{
			ForeignID:        strconv.Itoa(object.Id),
			Name:             object.Name,
			Active:           !object.Completed,
			ForeignProjectID: strconv.Itoa(object.ProjectId),
		}
		tasks = append(tasks, &task)
	}
	return tasks, nil
}

func (s *Service) ExportTimeEntry(t *toggl.TimeEntry) (int, error) {
	return 0, nil
}

func (s *Service) client() *basecamp.Client {
	return &basecamp.Client{
		ModifiedSince: s.modifiedSince,
		AccessToken:   s.token.AccessToken,
	}
}
