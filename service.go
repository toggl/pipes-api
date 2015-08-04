package main

import (
	"fmt"
	"time"
)

type (
	// Service interface for external services
	// Example implementation: github.go
	Service interface {
		// Name of the service
		Name() string

		// WorkspaceID helper function, should just return workspaceID
		WorkspaceID() int

		// setSince takes the provided time.Time
		// and adds it to Service struct. This can be used
		// to fetch just the modified data from external services.
		setSince(*time.Time)

		// setParams takes the necessary Service params
		// (for example the selected account id) as JSON
		// and adds them to Service struct.
		setParams([]byte) error

		// setAuthData adds the provided oauth token to Service struct
		setAuthData([]byte) error

		// keyFor should provide unique key for object type
		// Example: asana:account:XXXX:projects
		keyFor(string) string

		// Accounts maps foreign account to Account models
		// https://github.com/toggl/pipes-api/blob/master/model.go#L9-L12
		Accounts() ([]*Account, error)

		// Users maps foreign users to User models
		// https://github.com/toggl/pipes-api/blob/master/model.go#L14-L19
		Users() ([]*User, error)

		// Clients maps foreign clients to Client models
		// https://github.com/toggl/pipes-api/blob/master/model.go#L21-L25
		Clients() ([]*Client, error)

		// Projects maps foreign projects to Project models
		// https://github.com/toggl/pipes-api/blob/master/model.go#L27-L36
		Projects() ([]*Project, error)

		// Tasks maps foreign tasks to Task models
		// https://github.com/toggl/pipes-api/blob/master/model.go#L38-L45
		Tasks() ([]*Task, error)

		// TodoLists maps foreign todo lists to Task models
		// https://github.com/toggl/pipes-api/blob/master/model.go#L38-45
		TodoLists() ([]*Task, error)

		// Exports time entry model to foreign service
		// should return foreign id of saved time entry
		// https://github.com/toggl/pipes-api/blob/master/model.go#L47-L61
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
	case "github":
		return Service(&GithubService{workspaceID: workspaceID})
	case TestServiceName:
		return Service(&TestService{workspaceID: workspaceID})
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
