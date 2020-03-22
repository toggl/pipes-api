package integration

import (
	"time"

	"github.com/toggl/pipes-api/pkg/toggl"
)

type ID string

const (
	BaseCamp   ID = "basecamp"
	FreshBooks ID = "freshbooks"
	TeamWeek   ID = "teamweek"
	Asana      ID = "asana"
	GitHub     ID = "github"
)

type PipeID string

const (
	UsersPipe       PipeID = "users"
	ClientsPipe     PipeID = "clients"
	ProjectsPipe    PipeID = "projects"
	TasksPipe       PipeID = "tasks"
	TodoListsPipe   PipeID = "todolists"
	TodosPipe       PipeID = "todos"
	TimeEntriesPipe PipeID = "timeentries"
	AccountsPipe    PipeID = "accounts"
)

// Integration interface for external integrations
// Example implementation: github.go
//go:generate mockery -name Integration -case underscore -inpkg
type Integration interface {
	// ID returns an ID of the service
	ID() ID

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
	Accounts() ([]*toggl.Account, error)

	// Users maps foreign users to User models
	Users() ([]*toggl.User, error)

	// Clients maps foreign clients to Client models
	Clients() ([]*toggl.Client, error)

	// Projects maps foreign projects to Project models
	Projects() ([]*toggl.Project, error)

	// Tasks maps foreign tasks to Task models
	Tasks() ([]*toggl.Task, error)

	// TodoLists maps foreign to do lists to Task models
	TodoLists() ([]*toggl.Task, error)

	// ExportTimeEntry exports time entry model to foreign service
	// should return foreign id of saved time entry
	// Implemented only for "freshbook" integration
	ExportTimeEntry(*toggl.TimeEntry) (int, error)
}
