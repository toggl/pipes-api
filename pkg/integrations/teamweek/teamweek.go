package teamweek

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"code.google.com/p/goauth2/oauth"
	"github.com/toggl/go-teamweek"

	"github.com/toggl/pipes-api/pkg/toggl"
)

const ServiceName = "teamweek"

type Service struct {
	WorkspaceID int
	*TeamweekParams
	token oauth.Token
}

type TeamweekParams struct {
	AccountID int `json:"account_id"`
}

func (s *Service) Name() string {
	return ServiceName
}

func (s *Service) GetWorkspaceID() int {
	return s.WorkspaceID
}

func (s *Service) KeyFor(objectType string) string {
	if s.TeamweekParams == nil {
		return fmt.Sprintf("teamweek:account:%s", objectType)
	}
	return fmt.Sprintf("teamweek:account:%d:%s", s.AccountID, objectType)
}

func (s *Service) SetParams(b []byte) error {
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s.TeamweekParams == nil || s.AccountID == 0 {
		return errors.New("account_id must be present")
	}
	return nil
}

func (s *Service) SetAuthData(b []byte) error {
	if err := json.Unmarshal(b, &s.token); err != nil {
		return err
	}
	return nil
}

func (s *Service) client() *teamweek.Client {
	t := &oauth.Transport{Token: &s.token}
	return teamweek.NewClient(t.Client())
}

// Map Teamweek accounts to local accounts
func (s *Service) Accounts() ([]*toggl.Account, error) {
	foreignObject, err := s.client().GetUserProfile()
	if err != nil {
		return nil, err
	}
	var accounts []*toggl.Account
	for _, object := range foreignObject.Workspaces {
		account := toggl.Account{
			ID:   object.ID,
			Name: object.Name,
		}
		accounts = append(accounts, &account)
	}
	return accounts, nil
}

// Map Teamweek people to local users
func (s *Service) Users() ([]*toggl.User, error) {
	foreignObjects, err := s.client().ListWorkspaceMembers(int64(s.AccountID))
	if err != nil {
		return nil, err
	}
	var users []*toggl.User
	for _, object := range foreignObjects {
		if object.Dummy {
			continue
		}
		user := toggl.User{
			ForeignID: strconv.FormatInt(object.ID, 10),
			Name:      object.Name,
			Email:     object.Email,
		}
		users = append(users, &user)
	}
	return users, nil
}

// Map Teamweek projects to projects
func (s *Service) Projects() ([]*toggl.Project, error) {
	foreignObjects, err := s.client().ListWorkspaceProjects(int64(s.AccountID))
	if err != nil {
		return nil, err
	}
	var projects []*toggl.Project
	for _, object := range foreignObjects {
		project := toggl.Project{
			ForeignID: strconv.FormatInt(object.ID, 10),
			Name:      object.Name,
			Active:    true,
		}
		projects = append(projects, &project)
	}
	return projects, nil
}

// Map Teamweek tasks to tasks
func (s *Service) Tasks() ([]*toggl.Task, error) {
	foreignObjects, err := s.client().ListWorkspaceTasks(int64(s.AccountID))
	if err != nil {
		return nil, err
	}
	var tasks []*toggl.Task
	for _, object := range foreignObjects {
		if object.Project == nil {
			continue
		}
		task := toggl.Task{
			ForeignID:        strconv.FormatInt(object.ID, 10),
			Name:             object.Name,
			Active:           !object.Done,
			ForeignProjectID: strconv.FormatInt(object.ProjectID, 10),
		}
		tasks = append(tasks, &task)
	}
	return tasks, nil
}

func (s *Service) SetSince(*time.Time) {}

func (s *Service) TodoLists() ([]*toggl.Task, error) {
	return []*toggl.Task{}, nil
}

func (s *Service) Clients() ([]*toggl.Client, error) {
	return []*toggl.Client{}, nil
}

func (s *Service) ExportTimeEntry(*toggl.TimeEntry) (int, error) {
	return 0, nil
}
