package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/tambet/oauthplain"
	"github.com/toggl/go-freshbooks"
)

type FreshbooksService struct {
	emptyService
	workspaceID int
	accountName string
	token       oauthplain.Token
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
	return nil
}

func (s *FreshbooksService) setAuthData(b []byte) error {
	if err := json.Unmarshal(b, &s.token); err != nil {
		return err
	}
	s.accountName = s.token.Extra["account_name"]
	return nil
}

func (s *FreshbooksService) Accounts() ([]*Account, error) {
	return nil, nil
}

func (s *FreshbooksService) Api() *freshbooks.Api {
	return freshbooks.NewApi(s.accountName, &s.token)
}

func (s *FreshbooksService) Users() ([]*User, error) {
	foreignObjects, err := s.Api().Users()
	if err != nil {
		return nil, err
	}
	var users []*User
	for _, object := range foreignObjects {
		user := User{
			ForeignID: strconv.Itoa(object.UserId),
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
			ForeignID: strconv.Itoa(object.ClientId),
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
			Active:          true,
			Billable:        true,
			Name:            object.Name,
			ForeignID:       strconv.Itoa(object.ProjectId),
			foreignClientID: object.ClientId,
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

	tasksMap := make(map[int]freshbooks.Task)
	for _, task := range foreignTasks {
		tasksMap[task.TaskId] = task
	}

	var tasks []*Task
	for _, project := range foreignProjects {
		for _, taskID := range project.TaskIds {
			task := tasksMap[taskID]
			tasks = append(tasks, &Task{
				Active:           true,
				Name:             task.Name,
				ForeignID:        fmt.Sprintf("%d-%d", task.TaskId, project.ProjectId),
				foreignProjectID: strconv.Itoa(project.ProjectId),
			})
		}
	}
	return tasks, nil
}

func (s *FreshbooksService) ExportTimeEntry(t *TimeEntry) (int, error) {
	start, err := time.Parse(time.RFC3339, t.Start)
	if err != nil {
		return 0, err
	}
	entry := &freshbooks.TimeEntry{
		TimeEntryId: numberStrToInt(t.foreignID),
		ProjectId:   numberStrToInt(t.foreignProjectID),
		TaskId:      numberStrToInt(t.foreignTaskID),
		UserId:      numberStrToInt(t.foreignUserID),
		Hours:       float64(t.DurationInSeconds) / 3600,
		Notes:       t.Description,
		Date:        start.Format("2006-01-02"),
	}
	if entry.TaskId == 0 {
		return 0, fmt.Errorf("task not provided for time entry '%s'", entry.Notes)
	}
	return s.Api().SaveTimeEntry(entry)
}
