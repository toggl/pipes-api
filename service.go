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
		ForeignID int    `json:"foreign_id,omitempty"`
	}

	UsersResponse struct {
		Error string  `json:"error"`
		Users []*User `json:"users"`
	}

	Project struct {
		ID        int    `json:"id,omitempty"`
		Name      string `json:"name"`
		Active    bool   `json:"active"`
		ForeignID int    `json:"foreign_id,omitempty"`
	}

	Task struct {
		ID               int    `json:"id,omitempty"`
		Name             string `json:"name"`
		Active           bool   `json:"active"`
		ForeignID        int    `json:"foreign_id,omitempty"`
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

	Service interface {
		Name() string
		WorkspaceID() int
		setAuthData(*Authorization)
		setAccount(int)
		keyFor(string) string

		Users() ([]*User, error)
		Tasks() ([]*Task, error)
		TodoLists() ([]*Task, error)
		Projects() ([]*Project, error)
		Accounts() ([]*Account, error)
	}
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
