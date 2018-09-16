package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"code.google.com/p/goauth2/oauth"
	"github.com/fdelbos/go-teamweek/teamweek"
)

type TeamweekService struct {
	emptyService
	workspaceID int
	*TeamweekParams
	token oauth.Token
}

type TeamweekParams struct {
	AccountID int `json:"account_id"`
}

func (s *TeamweekService) Name() string {
	return "teamweek"
}

func (s *TeamweekService) WorkspaceID() int {
	return s.workspaceID
}

func (s *TeamweekService) keyFor(objectType string) string {
	if s.TeamweekParams == nil {
		return fmt.Sprintf("teamweek:account:%s", objectType)
	}
	return fmt.Sprintf("teamweek:account:%d:%s", s.AccountID, objectType)
}

func (s *TeamweekService) setParams(b []byte) error {
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s.TeamweekParams == nil || s.AccountID == 0 {
		return errors.New("account_id must be present")
	}
	return nil
}

func (s *TeamweekService) setAuthData(b []byte) error {
	if err := json.Unmarshal(b, &s.token); err != nil {
		return err
	}
	return nil
}

func (s *TeamweekService) client() *teamweek.Client {
	t := &oauth.Transport{Token: &s.token}
	return teamweek.NewClient(t.Client())
}

// Map Teamweek accounts to local accounts
func (s *TeamweekService) Accounts() ([]*Account, error) {
	foreignObject, err := s.client.GetUserProfile()
	if err != nil {
		return nil, err
	}
	var accounts []*Account
	for _, object := range foreignObject.Workspaces {
		account := Account{
			ID:   int(object.ID),
			Name: object.Name,
		}
		accounts = append(accounts, &account)
	}
	return accounts, nil
}

// Map Teamweek people to local users
func (s *TeamweekService) Users() ([]*User, error) {
	foreignObjects, err := s.client().ListWorkspaceMembers(int64(s.AccountID))
	if err != nil {
		return nil, err
	}
	var users []*User
	for _, object := range foreignObjects {
		if object.Dummy {
			continue
		}
		user := User{
			ForeignID: strconv.FormatInt(object.ID, 10),
			Name:      object.Name,
			Email:     object.Email,
		}
		users = append(users, &user)
	}
	return users, nil
}

// Map Teamweek projects to projects
func (s *TeamweekService) Projects() ([]*Project, error) {
	foreignObjects, err := s.client().ListWorkspaceProjects(int64(s.AccountID))
	if err != nil {
		return nil, err
	}
	var projects []*Project
	for _, object := range foreignObjects {
		project := Project{
			ForeignID: strconv.FormatInt(object.ID, 10),
			Name:      object.Name,
			Active:    true,
		}
		projects = append(projects, &project)
	}
	return projects, nil
}

// Map Teamweek tasks to tasks
func (s *TeamweekService) Tasks() ([]*Task, error) {
	foreignObjects, err := s.client().ListWorkspaceTasks(int64(s.AccountID))
	if err != nil {
		return nil, err
	}
	var tasks []*Task
	for _, object := range foreignObjects {
		if object.Project == nil {
			continue
		}
		task := Task{
			ForeignID:        strconv.FormatInt(object.ID, 10),
			Name:             object.Name,
			Active:           !object.Done,
			foreignProjectID: int(object.ProjectID),
		}
		tasks = append(tasks, &task)
	}
	return tasks, nil
}
