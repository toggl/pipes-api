package main

import (
	"code.google.com/p/goauth2/oauth"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tambet/go-asana/asana"
	"strconv"
)

type AsanaService struct {
	emptyService
	workspaceID int
	*AsanaParams
	token oauth.Token
}

type AsanaParams struct {
	AccountID int `json:"account_id"`
}

func (s *AsanaService) Name() string {
	return "asana"
}

func (s *AsanaService) WorkspaceID() int {
	return s.workspaceID
}

func (s *AsanaService) keyFor(objectType string) string {
	if s.AsanaParams == nil {
		return fmt.Sprintf("asana:account:%s", objectType)
	}
	return fmt.Sprintf("asana:account:%d:%s", s.AccountID, objectType)
}

func (s *AsanaService) setParams(b []byte) error {
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s.AsanaParams == nil || s.AccountID == 0 {
		return errors.New("account_id must be present")
	}
	return nil
}

func (s *AsanaService) setAuthData(b []byte) error {
	if err := json.Unmarshal(b, &s.token); err != nil {
		return err
	}
	return nil
}

func (s *AsanaService) client() *asana.Client {
	t := &oauth.Transport{Token: &s.token}
	return asana.NewClient(t.Client())
}

// Map Asana accounts to local accounts
func (s *AsanaService) Accounts() ([]*Account, error) {
	foreignObjects, err := s.client().ListWorkspaces()
	if err != nil {
		return nil, err
	}
	var accounts []*Account
	for _, object := range foreignObjects {
		account := Account{
			ID:   int(object.ID),
			Name: object.Name,
		}
		accounts = append(accounts, &account)
	}
	return accounts, nil
}

// Map Asana users to users
func (s *AsanaService) Users() ([]*User, error) {
	opt := &asana.Filter{Workspace: int64(s.AccountID)}
	foreignObjects, err := s.client().ListUsers(opt)
	if err != nil {
		return nil, err
	}
	var users []*User
	for _, object := range foreignObjects {
		user := User{
			ForeignID: strconv.FormatInt(object.ID, 10),
			Name:      object.Name,
			Email:     object.Email,
		}
		users = append(users, &user)
	}
	return users, nil
}

// Map Asana projects to projects
func (s *AsanaService) Projects() ([]*Project, error) {
	opt := &asana.Filter{Workspace: int64(s.AccountID)}
	foreignObjects, err := s.client().ListProjects(opt)
	if err != nil {
		return nil, err
	}
	var projects []*Project
	for _, object := range foreignObjects {
		project := Project{
			ForeignID: strconv.FormatInt(object.ID, 10),
			Name:      object.Name,
			Active:    !object.Archived,
		}
		projects = append(projects, &project)
	}
	return projects, nil
}

// Map Asana tasks to tasks
func (s *AsanaService) Tasks() ([]*Task, error) {
	opt := &asana.Filter{Workspace: int64(s.AccountID)}
	foreignProjects, err := s.client().ListProjects(opt)
	if err != nil {
		return nil, err
	}

	var tasks []*Task
	for _, project := range foreignProjects {
		opt.Project = project.ID
		foreignObjects, err := s.client().ListTasks(opt)
		if err != nil {
			return nil, err
		}
		for _, object := range foreignObjects {
			task := Task{
				ForeignID:        strconv.FormatInt(object.ID, 10),
				Name:             object.Name,
				Active:           !object.Completed,
				foreignProjectID: int(project.ID),
			}
			tasks = append(tasks, &task)
		}
	}
	return tasks, nil
}
