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
		ID              int    `json:"id,omitempty"`
		Name            string `json:"name"`
		Active          bool   `json:"active"`
		ClientID        int    `json:"cid"`
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
	}
	emptyService struct{}
)

func getService(serviceID string, workspaceID int) Service {
	switch serviceID {
	case "basecamp":
		return Service(&BasecampService{workspaceID: workspaceID})
	case "freshbooks":
		return Service(&FreshbooksService{workspaceID: workspaceID})
	default:
		panic(fmt.Sprintf("getService: Unrecognized serviceID - %s", serviceID))
	}
}

func (s *emptyService) Users() ([]*User, error)       { return nil, nil }
func (s *emptyService) Tasks() ([]*Task, error)       { return nil, nil }
func (s *emptyService) Clients() ([]*Client, error)   { return nil, nil }
func (s *emptyService) TodoLists() ([]*Task, error)   { return nil, nil }
func (s *emptyService) Projects() ([]*Project, error) { return nil, nil }
func (s *emptyService) Accounts() ([]*Account, error) { return nil, nil }
