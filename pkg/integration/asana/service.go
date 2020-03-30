package asana

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"code.google.com/p/goauth2/oauth"
	"github.com/bugsnag/bugsnag-go"
	"github.com/range-labs/go-asana/asana"

	"github.com/toggl/pipes-api/pkg/domain"
)

var asanaPerPageLimit uint32 = 100

type Service struct {
	WorkspaceID int
	*AsanaParams
	token oauth.Token
}

type AsanaParams struct {
	AccountID int64 `json:"account_id"`
}

func (s *Service) ID() domain.IntegrationID {
	return domain.Asana
}

func (s *Service) GetWorkspaceID() int {
	return s.WorkspaceID
}

func (s *Service) KeyFor(objectType domain.PipeID) string {
	if s.AsanaParams == nil {
		return fmt.Sprintf("asana:account:%s", objectType)
	}
	return fmt.Sprintf("asana:account:%d:%s", s.AccountID, objectType)
}

func (s *Service) SetParams(b []byte) error {
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s.AsanaParams == nil || s.AccountID == 0 {
		return errors.New("account_id must be present")
	}
	return nil
}

// SetAuthData sets Asana API Authentication Token.
// It should be `oauth.Token` structure from "code.google.com/p/goauth2/oauth"
func (s *Service) SetAuthData(b []byte) error {
	return json.Unmarshal(b, &s.token)
}

// Map Asana accounts to local accounts
func (s *Service) Accounts() ([]*domain.Account, error) {
	foreignObjects, err := s.client().ListWorkspaces(context.Background())
	if err != nil {
		bugsnag.Notify(err, bugsnag.MetaData{
			"asana_service": {
				"method":           "Accounts()",
				"remote_method":    "ListWorkspaces()",
				"asana_account_id": s.AccountID,
				"workspace_id":     s.GetWorkspaceID(),
			},
		})
		return nil, err
	}
	var accounts []*domain.Account
	for _, object := range foreignObjects {
		account := domain.Account{
			ID:   numberStrToInt64(object.GID),
			Name: object.Name,
		}
		accounts = append(accounts, &account)
	}
	return accounts, nil
}

// Map Asana users to users
func (s *Service) Users() ([]*domain.User, error) {
	opt := &asana.Filter{
		Workspace: s.AccountID,
		Limit:     asanaPerPageLimit,
	}
	foreignObjects, err := s.client().ListUsers(context.Background(), opt)
	if err != nil {
		bugsnag.Notify(err, bugsnag.MetaData{
			"asana_service": {
				"method":           "Users()",
				"remote_method":    "ListUsers()",
				"filter_workspace": s.AccountID,
				"asana_account_id": s.AccountID,
				"workspace_id":     s.GetWorkspaceID(),
			},
		})
		return nil, err
	}
	var users []*domain.User
	for _, object := range foreignObjects {
		user := domain.User{
			ForeignID: object.GID,
			Name:      object.Name,
			Email:     object.Email,
		}
		users = append(users, &user)
	}
	return users, nil
}

// Map Asana projects to projects
func (s *Service) Projects() ([]*domain.Project, error) {
	opt := &asana.Filter{
		Workspace: s.AccountID,
		Limit:     asanaPerPageLimit,
	}
	foreignObjects, err := s.client().ListProjects(context.Background(), opt)
	if err != nil {
		bugsnag.Notify(err, bugsnag.MetaData{
			"asana_service": {
				"method":           "Projects()",
				"remote_method":    "ListProjects()",
				"filter_workspace": s.AccountID,
				"asana_account_id": s.AccountID,
				"workspace_id":     s.GetWorkspaceID(),
			},
		})
		return nil, err
	}
	var projects []*domain.Project
	for _, object := range foreignObjects {
		project := domain.Project{
			ForeignID: object.GID,
			Name:      object.Name,
			Active:    !object.Archived,
		}
		projects = append(projects, &project)
	}
	return projects, nil
}

// Map Asana tasks to tasks
func (s *Service) Tasks() ([]*domain.Task, error) {
	opt := &asana.Filter{
		Workspace: s.AccountID,
		Limit:     asanaPerPageLimit,
	}
	foreignProjects, err := s.client().ListProjects(context.Background(), opt)
	if err != nil {
		bugsnag.Notify(err, bugsnag.MetaData{
			"asana_service": {
				"method":           "Tasks()",
				"remote_method":    "ListProjects()",
				"filter_workspace": s.AccountID,
				"asana_account_id": s.AccountID,
				"workspace_id":     s.GetWorkspaceID(),
			},
		})
		return nil, err
	}

	var tasks []*domain.Task
	for _, project := range foreignProjects {
		// list task only accept project filter
		opt := &asana.Filter{
			Project: numberStrToInt64(project.GID),
			Limit:   asanaPerPageLimit,
		}
		foreignObjects, err := s.client().ListTasks(context.Background(), opt)
		if err != nil {
			bugsnag.Notify(err, bugsnag.MetaData{
				"asana_service": {
					"method":           "Tasks()",
					"remote_method":    "ListTasks()",
					"filter_project":   project.GID,
					"asana_account_id": s.AccountID,
					"workspace_id":     s.GetWorkspaceID(),
				},
			})
			return nil, err
		}
		for _, object := range foreignObjects {
			task := domain.Task{
				ForeignID:        object.GID,
				Name:             object.Name,
				Active:           !object.Completed,
				ForeignProjectID: project.GID,
			}
			tasks = append(tasks, &task)
		}
	}
	return tasks, nil
}

func numberStrToInt64(s string) int64 {
	res, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return res
}

func (s *Service) SetSince(*time.Time) {}

func (s *Service) Clients() ([]*domain.Client, error) {
	return []*domain.Client{}, nil
}

func (s *Service) TodoLists() ([]*domain.Task, error) {
	return []*domain.Task{}, nil
}

func (s *Service) ExportTimeEntry(*domain.TimeEntry) (int, error) {
	return 0, nil
}

func (s *Service) client() *asana.Client {
	t := &oauth.Transport{Token: &s.token}
	return asana.NewClient(t.Client())
}
