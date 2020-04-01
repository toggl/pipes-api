package integration

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/tambet/oauthplain"
	"github.com/toggl/go-freshbooks"

	"github.com/toggl/pipes-api/pkg/domain"
)

type FreshBooksPipeIntegration struct {
	WorkspaceID int
	accountName string
	token       oauthplain.Token
}

func (s *FreshBooksPipeIntegration) ID() domain.IntegrationID {
	return domain.FreshBooks
}

func (s *FreshBooksPipeIntegration) GetWorkspaceID() int {
	return s.WorkspaceID
}

func (s *FreshBooksPipeIntegration) KeyFor(objectType domain.PipeID) string {
	return fmt.Sprintf("freshbooks:%s", objectType)
}

func (s *FreshBooksPipeIntegration) SetParams(b []byte) error {
	return nil
}

func (s *FreshBooksPipeIntegration) SetAuthData(b []byte) error {
	if err := json.Unmarshal(b, &s.token); err != nil {
		return err
	}
	s.accountName = s.token.Extra["account_name"]
	return nil
}

func (s *FreshBooksPipeIntegration) SetSince(*time.Time) {}

func (s *FreshBooksPipeIntegration) Users() ([]*domain.User, error) {
	foreignObjects, err := s.client().Users()
	if err != nil {
		return nil, err
	}
	var users []*domain.User
	for _, object := range foreignObjects {
		user := domain.User{
			ForeignID: strconv.Itoa(object.UserId),
			Name:      fmt.Sprintf("%s %s", object.FirstName, object.LastName),
			Email:     object.Email,
		}
		users = append(users, &user)
	}
	return users, nil
}

func (s *FreshBooksPipeIntegration) Clients() ([]*domain.Client, error) {
	foreignObjects, err := s.client().Clients()
	if err != nil {
		return nil, err
	}
	var clients []*domain.Client
	for _, object := range foreignObjects {
		client := domain.Client{
			ForeignID: strconv.Itoa(object.ClientId),
			Name:      object.Name,
		}
		clients = append(clients, &client)
	}
	return clients, nil
}

func (s *FreshBooksPipeIntegration) Projects() ([]*domain.Project, error) {
	foreignObjects, err := s.client().Projects()
	if err != nil {
		return nil, err
	}
	var projects []*domain.Project
	for _, object := range foreignObjects {
		project := domain.Project{
			Active:          true,
			Billable:        true,
			Name:            object.Name,
			ForeignID:       strconv.Itoa(object.ProjectId),
			ForeignClientID: object.ClientId,
		}
		projects = append(projects, &project)
	}
	return projects, nil
}

func (s *FreshBooksPipeIntegration) Tasks() ([]*domain.Task, error) {
	foreignProjects, err := s.client().Projects()
	if err != nil {
		return nil, err
	}
	foreignTasks, err := s.client().Tasks()
	if err != nil {
		return nil, err
	}

	tasksMap := make(map[int]freshbooks.Task)
	for _, task := range foreignTasks {
		tasksMap[task.TaskId] = task
	}

	var tasks []*domain.Task
	for _, project := range foreignProjects {
		for _, taskID := range project.TaskIds {
			task := tasksMap[taskID]
			tasks = append(tasks, &domain.Task{
				Active:           true,
				Name:             task.Name,
				ForeignID:        fmt.Sprintf("%d-%d", task.TaskId, project.ProjectId),
				ForeignProjectID: strconv.Itoa(project.ProjectId),
			})
		}
	}
	return tasks, nil
}

func (s *FreshBooksPipeIntegration) ExportTimeEntry(t *domain.TimeEntry) (int, error) {
	start, err := time.Parse(time.RFC3339, t.Start)
	if err != nil {
		return 0, err
	}
	entry := &freshbooks.TimeEntry{
		TimeEntryId: numberStrToInt(t.ForeignID),
		ProjectId:   numberStrToInt(t.ForeignProjectID),
		TaskId:      numberStrToInt(t.ForeignTaskID),
		UserId:      numberStrToInt(t.ForeignUserID),
		Hours:       float64(t.DurationInSeconds) / 3600,
		Notes:       t.Description,
		Date:        start.Format("2006-01-02"),
	}
	if entry.TaskId == 0 {
		return 0, fmt.Errorf("task not provided for time entry '%s'", entry.Notes)
	}
	return s.client().SaveTimeEntry(entry)
}

func (s *FreshBooksPipeIntegration) Accounts() ([]*domain.Account, error) {
	return []*domain.Account{}, nil
}

func (s *FreshBooksPipeIntegration) TodoLists() ([]*domain.Task, error) {
	return []*domain.Task{}, nil
}

func (s *FreshBooksPipeIntegration) client() *freshbooks.Api {
	return freshbooks.NewApi(s.accountName, &s.token)
}

func numberStrToInt(s string) int {
	res, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return res
}
