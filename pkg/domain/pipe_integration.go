package domain

import (
	"time"
)

// Integration interface for external integrations
// Example implementation: github.go
//go:generate mockery -name PipeIntegration -case underscore -outpkg mocks
type PipeIntegration interface {
	// IntegrationID returns an IntegrationID of the service
	ID() IntegrationID

	// GetWorkspaceID helper function, should just return workspaceID
	GetWorkspaceID() int

	// SetSince takes the provided time.Time
	// and adds it to Integration struct. This can be used
	// to fetch just the modified data from external services.
	// Implemented only for "basecamp" integration.
	SetSince(*time.Time)

	// SetParams takes the necessary Integration params
	// (for example the selected account id) as JSON
	// and adds them to Integration struct.
	SetParams([]byte) error

	// SetAuthData adds the provided oauth token to Integration struct
	SetAuthData([]byte) error

	// KeyFor should provide unique key for object type
	// Example: asana:account:XXXX:projects
	KeyFor(PipeID) string

	// Accounts maps foreign account to Account models
	Accounts() ([]*Account, error)

	// Users maps foreign users to User models
	Users() ([]*User, error)

	// Clients maps foreign clients to Client models
	Clients() ([]*Client, error)

	// Projects maps foreign projects to Project models
	Projects() ([]*Project, error)

	// Tasks maps foreign tasks to Task models
	Tasks() ([]*Task, error)

	// TodoLists maps foreign to do lists to Task models
	TodoLists() ([]*Task, error)

	// ExportTimeEntry exports time entry model to foreign service
	// should return foreign id of saved time entry
	// Implemented only for "freshbook" integration
	ExportTimeEntry(*TimeEntry) (int, error)
}
