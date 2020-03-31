package integration

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"code.google.com/p/goauth2/oauth"
	"github.com/toggl/go-teamweek"

	"github.com/toggl/pipes-api/pkg/domain"
)

type TogglPlanPipeIntegration struct {
	WorkspaceID int
	*Params
	token oauth.Token
}

type Params struct {
	AccountID int `json:"account_id"`
}

func (s *TogglPlanPipeIntegration) ID() domain.IntegrationID {
	return domain.TogglPlan
}

func (s *TogglPlanPipeIntegration) GetWorkspaceID() int {
	return s.WorkspaceID
}

func (s *TogglPlanPipeIntegration) KeyFor(objectType domain.PipeID) string {
	if s.Params == nil {
		return fmt.Sprintf("%s:account:%s", domain.TogglPlan, objectType)
	}
	return fmt.Sprintf("%s:account:%d:%s", domain.TogglPlan, s.AccountID, objectType)
}

func (s *TogglPlanPipeIntegration) SetParams(b []byte) error {
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s.Params == nil || s.AccountID == 0 {
		return errors.New("account_id must be present")
	}
	return nil
}

func (s *TogglPlanPipeIntegration) SetAuthData(b []byte) error {
	if err := json.Unmarshal(b, &s.token); err != nil {
		return err
	}
	return nil
}

// Map Teamweek accounts to local accounts
func (s *TogglPlanPipeIntegration) Accounts() ([]*domain.Account, error) {
	foreignObject, err := s.client().GetUserProfile()
	if err != nil {
		return nil, err
	}
	var accounts []*domain.Account
	for _, object := range foreignObject.Workspaces {
		account := domain.Account{
			ID:   object.ID,
			Name: object.Name,
		}
		accounts = append(accounts, &account)
	}
	return accounts, nil
}

// Map Teamweek people to local users
func (s *TogglPlanPipeIntegration) Users() ([]*domain.User, error) {
	foreignObjects, err := s.client().ListWorkspaceMembers(int64(s.AccountID))
	if err != nil {
		return nil, err
	}
	var users []*domain.User
	for _, object := range foreignObjects {
		if object.Dummy {
			continue
		}
		user := domain.User{
			ForeignID: strconv.FormatInt(object.ID, 10),
			Name:      object.Name,
			Email:     object.Email,
		}
		users = append(users, &user)
	}
	return users, nil
}

// Map Teamweek projects to projects
func (s *TogglPlanPipeIntegration) Projects() ([]*domain.Project, error) {
	foreignObjects, err := s.client().ListWorkspaceProjects(int64(s.AccountID))
	if err != nil {
		return nil, err
	}
	var projects []*domain.Project
	for _, object := range foreignObjects {
		project := domain.Project{
			ForeignID: strconv.FormatInt(object.ID, 10),
			Name:      object.Name,
			Active:    true,
		}
		projects = append(projects, &project)
	}
	return projects, nil
}

// Map Teamweek tasks to tasks
func (s *TogglPlanPipeIntegration) Tasks() ([]*domain.Task, error) {
	foreignObjects, err := s.client().ListWorkspaceTasks(int64(s.AccountID))
	if err != nil {
		return nil, err
	}
	var tasks []*domain.Task
	for _, object := range foreignObjects {
		if object.Project == nil {
			continue
		}
		task := domain.Task{
			ForeignID:        strconv.FormatInt(object.ID, 10),
			Name:             object.Name,
			Active:           !object.Done,
			ForeignProjectID: strconv.FormatInt(object.ProjectID, 10),
		}
		tasks = append(tasks, &task)
	}
	return tasks, nil
}

func (s *TogglPlanPipeIntegration) SetSince(*time.Time) {}

func (s *TogglPlanPipeIntegration) TodoLists() ([]*domain.Task, error) {
	return []*domain.Task{}, nil
}

func (s *TogglPlanPipeIntegration) Clients() ([]*domain.Client, error) {
	return []*domain.Client{}, nil
}

func (s *TogglPlanPipeIntegration) ExportTimeEntry(*domain.TimeEntry) (int, error) {
	return 0, nil
}

func (s *TogglPlanPipeIntegration) client() *teamweek.Client {
	t := &oauth.Transport{Token: &s.token}
	return teamweek.NewClient(t.Client())
}
