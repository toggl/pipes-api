package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"code.google.com/p/goauth2/oauth"
	"github.com/toggl/go-basecamp"
)

type BasecampService struct {
	emptyService
	workspaceID int
	*BasecampParams
	token         oauth.Token
	modifiedSince *time.Time
}

type BasecampParams struct {
	AccountID int `json:"account_id"`
}

func (s *BasecampService) Name() string {
	return "basecamp"
}

func (s *BasecampService) WorkspaceID() int {
	return s.workspaceID
}

func (s *BasecampService) keyFor(objectType string) string {
	if s.BasecampParams == nil {
		return fmt.Sprintf("basecamp:account:%s", objectType)
	}
	return fmt.Sprintf("basecamp:account:%d:%s", s.AccountID, objectType)
}

func (s *BasecampService) setParams(b []byte) error {
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s.BasecampParams == nil || s.AccountID == 0 {
		return errors.New("account_id must be present")
	}
	return nil
}

func (s *BasecampService) setAuthData(b []byte) error {
	return json.Unmarshal(b, &s.token)
}

func (s *BasecampService) setSince(since *time.Time) {
	s.modifiedSince = since
}

func (s *BasecampService) client() *basecamp.Client {
	return &basecamp.Client{
		ModifiedSince: s.modifiedSince,
		AccessToken:   s.token.AccessToken,
	}
}

// Map basecamp accounts to local accounts
func (s *BasecampService) Accounts() ([]*Account, error) {
	foreignObjects, err := s.client().GetAccounts()
	if err != nil {
		return nil, err
	}
	var accounts []*Account
	for _, object := range foreignObjects {
		account := Account{
			ID:   int64(object.Id),
			Name: object.Name,
		}
		accounts = append(accounts, &account)
	}
	return accounts, nil
}

// Map basecamp people to local users
func (s *BasecampService) Users() ([]*User, error) {
	foreignObjects, err := s.client().GetPeople(s.AccountID)
	if err != nil {
		return nil, err
	}
	var users []*User
	for _, object := range foreignObjects {
		user := User{
			ForeignID: strconv.Itoa(object.Id),
			Name:      object.Name,
			Email:     object.Email,
		}
		users = append(users, &user)
	}
	return users, nil
}

// Map basecamp projects to projects
func (s *BasecampService) Projects() ([]*Project, error) {
	foreignObjects, err := s.client().GetProjects(s.AccountID)
	if err != nil {
		return nil, err
	}
	var projects []*Project
	for _, object := range foreignObjects {
		if object.UpdatedAt.Before(*s.modifiedSince) {
			continue
		}
		project := Project{
			Active:    true,
			ForeignID: strconv.Itoa(object.Id),
			Name:      object.Name,
		}
		projects = append(projects, &project)
	}
	return projects, nil
}

// Map basecamp todos to tasks
func (s *BasecampService) Tasks() ([]*Task, error) {
	c := s.client()
	foreignObjects, err := c.GetAllTodoLists(s.AccountID)
	if err != nil {
		return nil, err
	}
	var tasks []*Task
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
			task := Task{
				ForeignID:        strconv.Itoa(todo.Id),
				Name:             fmt.Sprintf("[%s] %s", object.Name, todo.Content),
				Active:           true,
				foreignProjectID: strconv.Itoa(object.ProjectId),
			}
			tasks = append(tasks, &task)
		}
		for _, todo := range todoList.Todos.Completed {
			// if todo.UpdatedAt.Before(*s.modifiedSince) {
			// 	continue
			// }
			task := Task{
				ForeignID:        strconv.Itoa(todo.Id),
				Name:             fmt.Sprintf("[%s] %s", object.Name, todo.Content),
				Active:           false,
				foreignProjectID: strconv.Itoa(object.ProjectId),
			}
			tasks = append(tasks, &task)
		}
	}
	return tasks, nil
}

// Map basecamp todolists to tasks
func (s *BasecampService) TodoLists() ([]*Task, error) {
	foreignObjects, err := s.client().GetAllTodoLists(s.AccountID)
	if err != nil {
		return nil, err
	}
	var tasks []*Task
	for _, object := range foreignObjects {
		if object.UpdatedAt.Before(*s.modifiedSince) {
			continue
		}
		task := Task{
			ForeignID:        strconv.Itoa(object.Id),
			Name:             object.Name,
			Active:           !object.Completed,
			foreignProjectID: strconv.Itoa(object.ProjectId),
		}
		tasks = append(tasks, &task)
	}
	return tasks, nil
}

func (s *BasecampService) ExportTimeEntry(t *TimeEntry) (int, error) {
	return 0, nil
}
