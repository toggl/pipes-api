package main

import (
	"fmt"
	"time"
)

type (
	// Service interface for external services
	Service interface {
		// Name of the service
		Name() string
		WorkspaceID() int
		setSince(*time.Time)
		setParams([]byte) error
		setAuthData([]byte) error
		keyFor(string) string

		// Accounts maps foreign account to Account models
		// https://github.com/toggl/pipes-api/blob/master/model.go#L9
		Accounts() ([]*Account, error)

		// Users maps foreign users to User models
		// https://github.com/toggl/pipes-api/blob/master/model.go#L14
		Users() ([]*User, error)

		// Clients maps foreign clients to Client models
		// https://github.com/toggl/pipes-api/blob/master/model.go#L21
		Clients() ([]*Client, error)

		// Projects maps foreign projects to Project models
		// https://github.com/toggl/pipes-api/blob/master/model.go#L27
		Projects() ([]*Project, error)

		// Tasks maps foreign tasks to Task models
		// https://github.com/toggl/pipes-api/blob/master/model.go#L38
		Tasks() ([]*Task, error)

		// TodoLists maps foreign todo lists to Task models
		// https://github.com/toggl/pipes-api/blob/master/model.go#L38
		TodoLists() ([]*Task, error)

		// Exports time entry model to foreign service
		// https://github.com/toggl/pipes-api/blob/master/model.go#L47
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
func (s *emptyService) setParams([]byte) error                  { return nil }
func (s *emptyService) Users() ([]*User, error)                 { return nil, nil }
func (s *emptyService) Tasks() ([]*Task, error)                 { return nil, nil }
func (s *emptyService) Clients() ([]*Client, error)             { return nil, nil }
func (s *emptyService) TodoLists() ([]*Task, error)             { return nil, nil }
func (s *emptyService) Projects() ([]*Project, error)           { return nil, nil }
func (s *emptyService) Accounts() ([]*Account, error)           { return nil, nil }
func (s *emptyService) ExportTimeEntry(*TimeEntry) (int, error) { return 0, nil }
