package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"code.google.com/p/goauth2/oauth"
	"github.com/bugsnag/bugsnag-go"
	"github.com/range-labs/go-asana/asana"
)

type AsanaService struct {
	emptyService
	workspaceID int
	*AsanaParams
	token oauth.Token
}

type AsanaParams struct {
	AccountID int64 `json:"account_id"`
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
	return json.Unmarshal(b, &s.token)
}

func (s *AsanaService) client() *asana.Client {
	t := &oauth.Transport{Token: &s.token}
	return asana.NewClient(t.Client())
}

// Map Asana accounts to local accounts
func (s *AsanaService) Accounts() ([]*Account, error) {
	foreignObjects, err := s.client().ListWorkspaces(context.Background())
	if err != nil {
		bugsnag.Notify(err, bugsnag.MetaData{
			"asana_service": {
				"method":        "Accounts()",
				"remote_method": "ListWorkspaces()",
			},
		})
		return nil, err
	}
	var accounts []*Account
	for _, object := range foreignObjects {
		account := Account{
			ID:   numberStrToInt64(object.GID),
			Name: object.Name,
		}
		accounts = append(accounts, &account)
	}
	return accounts, nil
}

// Map Asana users to users
func (s *AsanaService) Users() ([]*User, error) {
	opt := &asana.Filter{Workspace: s.AccountID}
	foreignObjects, err := s.client().ListUsers(context.Background(), opt)
	if err != nil {
		bugsnag.Notify(err, bugsnag.MetaData{
			"asana_service": {
				"method":           "Users()",
				"remote_method":    "ListUsers()",
				"filter_workspace": s.AccountID,
			},
		})
		return nil, err
	}
	var users []*User
	for _, object := range foreignObjects {
		user := User{
			ForeignID: object.GID,
			Name:      object.Name,
			Email:     object.Email,
		}
		users = append(users, &user)
	}
	return users, nil
}

// Map Asana projects to projects
func (s *AsanaService) Projects() ([]*Project, error) {
	opt := &asana.Filter{Workspace: s.AccountID}
	foreignObjects, err := s.client().ListProjects(context.Background(), opt)
	if err != nil {
		bugsnag.Notify(err, bugsnag.MetaData{
			"asana_service": {
				"method":           "Projects()",
				"remote_method":    "ListProjects()",
				"filter_workspace": s.AccountID,
			},
		})
		return nil, err
	}
	var projects []*Project
	for _, object := range foreignObjects {
		project := Project{
			ForeignID: object.GID,
			Name:      object.Name,
			Active:    !object.Archived,
		}
		projects = append(projects, &project)
	}
	return projects, nil
}

// Map Asana tasks to tasks
func (s *AsanaService) Tasks() ([]*Task, error) {
	opt := &asana.Filter{Workspace: s.AccountID}
	foreignProjects, err := s.client().ListProjects(context.Background(), opt)
	if err != nil {
		bugsnag.Notify(err, bugsnag.MetaData{
			"asana_service": {
				"method":           "Tasks()",
				"remote_method":    "ListProjects()",
				"filter_workspace": s.AccountID,
			},
		})
		return nil, err
	}

	var tasks []*Task
	for _, project := range foreignProjects {
		// list task only accept project filter
		opt := &asana.Filter{
			Project: numberStrToInt64(project.GID),
		}
		foreignObjects, err := s.client().ListTasks(context.Background(), opt)
		if err != nil {
			bugsnag.Notify(err, bugsnag.MetaData{
				"asana_service": {
					"method":         "Tasks()",
					"remote_method":  "ListTasks()",
					"filter_project": project.GID,
				},
			})
			return nil, err
		}
		for _, object := range foreignObjects {
			task := Task{
				ForeignID:        object.GID,
				Name:             object.Name,
				Active:           !object.Completed,
				foreignProjectID: project.GID,
			}
			tasks = append(tasks, &task)
		}
	}
	return tasks, nil
}
