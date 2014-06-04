package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tambet/oauthplain"
	"github.com/toggl/go-freshbooks"
	"strconv"
)

type FreshbooksService struct {
	emptyService
	workspaceID int
	*FreshbooksParams
	token oauthplain.Token
}

type FreshbooksParams struct {
	AccountName string `json:"account_name"`
}

func (s *FreshbooksService) Name() string {
	return "freshbooks"
}

func (s *FreshbooksService) WorkspaceID() int {
	return s.workspaceID
}

func (s *FreshbooksService) keyFor(objectType string) string {
	return fmt.Sprintf("freshbooks:%s", objectType)
}

func (s *FreshbooksService) setParams(b []byte) error {
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s.AccountName == "" {
		return errors.New("account_name must be present")
	}
	return nil
}

func (s *FreshbooksService) setAuthData(b []byte) error {
	if err := json.Unmarshal(b, &s.token); err != nil {
		return err
	}
	return nil
}

func (s *FreshbooksService) Accounts() ([]*Account, error) {
	return nil, nil
}

func (s *FreshbooksService) Api() *freshbooks.Api {
	return freshbooks.NewApi(s.AccountName, &s.token)
}

func (s *FreshbooksService) Users() ([]*User, error) {
	foreignObjects, err := s.Api().Users()
	if err != nil {
		return nil, err
	}
	var users []*User
	for _, object := range foreignObjects {
		user := User{
			ForeignID: object.UserId,
			Name:      fmt.Sprintf("%s %s", object.FirstName, object.LastName),
			Email:     object.Email,
		}
		users = append(users, &user)
	}
	return users, nil
}

func (s *FreshbooksService) Clients() ([]*Client, error) {
	foreignObjects, err := s.Api().Clients()
	if err != nil {
		return nil, err
	}
	var clients []*Client
	for _, object := range foreignObjects {
		client := Client{
			ForeignID: object.ClientId,
			Name:      object.Name,
		}
		clients = append(clients, &client)
	}
	return clients, nil
}

func (s *FreshbooksService) Projects() ([]*Project, error) {
	foreignObjects, err := s.Api().Projects()
	if err != nil {
		return nil, err
	}
	var projects []*Project
	for _, object := range foreignObjects {
		project := Project{
			Name:            object.Name,
			ForeignID:       object.ProjectId,
			foreignClientID: convertInt(object.ClientId),
		}
		projects = append(projects, &project)
	}
	return projects, nil
}

func (s *FreshbooksService) Tasks() ([]*Task, error) {
	foreignProjects, err := s.Api().Projects()
	if err != nil {
		return nil, err
	}
	foreignTasks, err := s.Api().Tasks()
	if err != nil {
		return nil, err
	}

	tasksMap := make(map[int]*freshbooks.Task)
	for _, task := range foreignTasks {
		tasksMap[task.TaskId] = &task
	}

	var tasks []*Task
	for _, project := range foreignProjects {
		for _, taskID := range project.TaskIds {
			task := tasksMap[taskID]
			tasks = append(tasks, &Task{
				Name:             task.Name,
				ForeignID:        task.TaskId,
				foreignProjectID: project.ProjectId,
			})
		}
	}
	return tasks, nil
}

func convertInt(s string) int {
	if s == "" {
		return 0
	}
	res, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return res
}
