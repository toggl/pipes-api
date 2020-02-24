package freshbooks

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/tambet/oauthplain"
	"github.com/toggl/go-freshbooks"

	"github.com/toggl/pipes-api/pkg/toggl"
)

type Service struct {
	WorkspaceID int
	accountName string
	token       oauthplain.Token
}

func (s *Service) Name() string {
	return "freshbooks"
}

func (s *Service) GetWorkspaceID() int {
	return s.WorkspaceID
}

func (s *Service) KeyFor(objectType string) string {
	return fmt.Sprintf("freshbooks:%s", objectType)
}

func (s *Service) SetParams(b []byte) error {
	return nil
}

func (s *Service) SetAuthData(b []byte) error {
	if err := json.Unmarshal(b, &s.token); err != nil {
		return err
	}
	s.accountName = s.token.Extra["account_name"]
	return nil
}

func (s *Service) Accounts() ([]*toggl.Account, error) {
	return nil, nil
}

func (s *Service) Api() *freshbooks.Api {
	return freshbooks.NewApi(s.accountName, &s.token)
}

func (s *Service) Users() ([]*toggl.User, error) {
	foreignObjects, err := s.Api().Users()
	if err != nil {
		return nil, err
	}
	var users []*toggl.User
	for _, object := range foreignObjects {
		user := toggl.User{
			ForeignID: strconv.Itoa(object.UserId),
			Name:      fmt.Sprintf("%s %s", object.FirstName, object.LastName),
			Email:     object.Email,
		}
		users = append(users, &user)
	}
	return users, nil
}

func (s *Service) Clients() ([]*toggl.Client, error) {
	foreignObjects, err := s.Api().Clients()
	if err != nil {
		return nil, err
	}
	var clients []*toggl.Client
	for _, object := range foreignObjects {
		client := toggl.Client{
			ForeignID: strconv.Itoa(object.ClientId),
			Name:      object.Name,
		}
		clients = append(clients, &client)
	}
	return clients, nil
}

func (s *Service) Projects() ([]*toggl.Project, error) {
	foreignObjects, err := s.Api().Projects()
	if err != nil {
		return nil, err
	}
	var projects []*toggl.Project
	for _, object := range foreignObjects {
		project := toggl.Project{
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

func (s *Service) Tasks() ([]*toggl.Task, error) {
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

	var tasks []*toggl.Task
	for _, project := range foreignProjects {
		for _, taskID := range project.TaskIds {
			task := tasksMap[taskID]
			tasks = append(tasks, &toggl.Task{
				Active:           true,
				Name:             task.Name,
				ForeignID:        fmt.Sprintf("%d-%d", task.TaskId, project.ProjectId),
				ForeignProjectID: strconv.Itoa(project.ProjectId),
			})
		}
	}
	return tasks, nil
}

func (s *Service) ExportTimeEntry(t *toggl.TimeEntry) (int, error) {
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
	return s.Api().SaveTimeEntry(entry)
}

func numberStrToInt(s string) int {
	res, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return res
}

func (s *Service) SetSince(*time.Time) {}

func (s *Service) TodoLists() ([]*toggl.Task, error) {
	return []*toggl.Task{}, nil
}
