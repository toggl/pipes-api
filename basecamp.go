package main

import (
	"code.google.com/p/goauth2/oauth"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/toggl/go-basecamp"
)

type BasecampService struct {
	workspaceID int
	*BasecampParams
	token oauth.Token
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
	if s.AccountID == 0 {
		return errors.New("account_id must be present")
	}
	return nil
}

func (s *BasecampService) setAuthData(b []byte) error {
	if err := json.Unmarshal(b, &s.token); err != nil {
		return err
	}
	return nil
}

func (s *BasecampService) client() *basecamp.Client {
	return &basecamp.Client{AccessToken: s.token.AccessToken}
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
			ID:   object.Id,
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
			ForeignID: object.Id,
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
		project := Project{
			Active:    true,
			ForeignID: object.Id,
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
	for _, object := range foreignObjects {
		todoList, err := c.GetTodoList(s.AccountID, object.ProjectId, object.Id)
		if err != nil {
			return nil, err
		}
		for _, todo := range todoList.Todos.Remaining {
			task := Task{
				ForeignID:        todo.Id,
				Name:             fmt.Sprintf("[%s] %s", object.Name, todo.Content),
				Active:           true,
				foreignProjectID: object.ProjectId,
			}
			tasks = append(tasks, &task)
		}
		for _, todo := range todoList.Todos.Completed {
			task := Task{
				ForeignID:        todo.Id,
				Name:             fmt.Sprintf("[%s] %s", object.Name, todo.Content),
				Active:           false,
				foreignProjectID: object.ProjectId,
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
		task := Task{
			ForeignID:        object.Id,
			Name:             object.Name,
			Active:           !object.Completed,
			foreignProjectID: object.ProjectId,
		}
		tasks = append(tasks, &task)
	}
	return tasks, nil
}
