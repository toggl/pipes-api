package main

import (
	"fmt"
	"github.com/toggl/go-basecamp"
)

type BasecampService struct {
	workspaceID int
	AccountID   int
	AccessToken string
}

func (s *BasecampService) Name() string {
	return "basecamp"
}

func (s *BasecampService) WorkspaceID() int {
	return s.workspaceID
}

func (s *BasecampService) keyFor(objectType string) string {
	return fmt.Sprintf("basecamp:account:%d:%s", s.AccountID, objectType)
}

func (s *BasecampService) setAuthData(a *Authorization) {
	s.AccessToken = a.AccessToken
}

func (s *BasecampService) setAccount(accountID int) {
	s.AccountID = accountID
}

// Map basecamp accounts to local accounts
func (s *BasecampService) Accounts() ([]*Account, error) {
	c := basecamp.Client{AccessToken: s.AccessToken}
	foreignObjects, err := c.GetAccounts()
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
	c := basecamp.Client{AccessToken: s.AccessToken}
	foreignObjects, err := c.GetPeople(s.AccountID)
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
	c := basecamp.Client{AccessToken: s.AccessToken}
	foreignObjects, err := c.GetProjects(s.AccountID)
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
	c := basecamp.Client{AccessToken: s.AccessToken}
	foreignObjects, err := c.GetTodoLists(s.AccountID)
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
	}
	return tasks, nil
}

// Map basecamp todolists to tasks
func (s *BasecampService) TodoLists() ([]*Task, error) {
	c := basecamp.Client{AccessToken: s.AccessToken}
	foreignObjects, err := c.GetTodoLists(s.AccountID)
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
