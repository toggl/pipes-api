package main

import (
	"fmt"
	"time"
)

type (
	Service interface {
		Name() string
		WorkspaceID() int
		setSince(*time.Time)
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

func (s *emptyService) setSince(*time.Time)                     {}
func (s *emptyService) Users() ([]*User, error)                 { return nil, nil }
func (s *emptyService) Tasks() ([]*Task, error)                 { return nil, nil }
func (s *emptyService) Clients() ([]*Client, error)             { return nil, nil }
func (s *emptyService) TodoLists() ([]*Task, error)             { return nil, nil }
func (s *emptyService) Projects() ([]*Project, error)           { return nil, nil }
func (s *emptyService) Accounts() ([]*Account, error)           { return nil, nil }
func (s *emptyService) ExportTimeEntry(*TimeEntry) (int, error) { return 0, nil }
