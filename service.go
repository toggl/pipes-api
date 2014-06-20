package main

import (
	"fmt"
)

type (
	Workspace struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	Account struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	AccountsResponse struct {
		Error    string     `json:"error"`
		Accounts []*Account `json:"accounts"`
	}

	User struct {
		ID        int    `json:"id,omitempty"`
		Email     string `json:"email"`
		Name      string `json:"name"`
		ForeignID string `json:"foreign_id,omitempty"`
	}

	UsersResponse struct {
		Error string  `json:"error"`
		Users []*User `json:"users"`
	}

	Client struct {
		ID        int    `json:"id,omitempty"`
		Name      string `json:"name"`
		ForeignID string `json:"foreign_id,omitempty"`
	}

	Project struct {
		ID       int    `json:"id,omitempty"`
		Name     string `json:"name,omitempty"`
		Active   bool   `json:"active,omitempty"`
		Billable bool   `json:"billable,omitempty"`
		ClientID int    `json:"cid,omitempty"`

		ForeignID       string `json:"foreign_id,omitempty"`
		foreignClientID int
	}

	Task struct {
		ID               int    `json:"id,omitempty"`
		Name             string `json:"name"`
		Active           bool   `json:"active"`
		ForeignID        string `json:"foreign_id,omitempty"`
		ProjectID        int    `json:"pid"`
		foreignProjectID int
	}

	TimeEntry struct {
		ID                int    `json:"id"`
		ProjectID         int    `json:"pid,omitempty"`
		TaskID            int    `json:"tid,omitempty"`
		UserID            int    `json:"uid,omitempty"`
		Billable          bool   `json:"billable"`
		Start             string `json:"start"`
		Stop              string `json:"stop,omitempty"`
		DurationInSeconds int    `json:"duration"`
		Description       string `json:"description,omitempty"`
		foreignID         int
		foreignTaskID     int
		foreignUserID     int
		foreignProjectID  int
	}

	ProjectsResponse struct {
		Error    string     `json:"error"`
		Projects []*Project `json:"projects"`
	}

	TasksResponse struct {
		Error string  `json:"error"`
		Tasks []*Task `json:"tasks"`
	}

	ClientsResponse struct {
		Error   string    `json:"error"`
		Clients []*Client `json:"clients"`
	}

	Service interface {
		Name() string
		WorkspaceID() int
		setParams([]byte) error
		setAuthData([]byte) error
		keyFor(string) string

		Users() ([]*User, error)
		Tasks() ([]*Task, error)
		Clients() ([]*Client, error)
		TodoLists() ([]*Task, error)
		Projects() ([]*Project, error)
		Accounts() ([]*Account, error)
		ExportTimeEntry(*TimeEntry) (int, error)
	}
	emptyService struct{}
)

func getService(serviceID string, workspaceID int) Service {
	switch serviceID {
	case "basecamp":
		return Service(&BasecampService{workspaceID: workspaceID})
	case "freshbooks":
		return Service(&FreshbooksService{workspaceID: workspaceID})
	case "teamweek":
		return Service(&TeamweekService{workspaceID: workspaceID})
	case "asana":
		return Service(&AsanaService{workspaceID: workspaceID})
	default:
		panic(fmt.Sprintf("getService: Unrecognized serviceID - %s", serviceID))
	}
}

func (s *emptyService) Users() ([]*User, error)                 { return nil, nil }
func (s *emptyService) Tasks() ([]*Task, error)                 { return nil, nil }
func (s *emptyService) Clients() ([]*Client, error)             { return nil, nil }
func (s *emptyService) TodoLists() ([]*Task, error)             { return nil, nil }
func (s *emptyService) Projects() ([]*Project, error)           { return nil, nil }
func (s *emptyService) Accounts() ([]*Account, error)           { return nil, nil }
func (s *emptyService) ExportTimeEntry(*TimeEntry) (int, error) { return 0, nil }
