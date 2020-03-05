package integrations

import (
	"time"

	"github.com/toggl/pipes-api/pkg/toggl"
)

// ExternalService interface for external integrations
// Example implementation: github.go
//go:generate mockery -name ExternalService -case underscore -inpkg
type ExternalService interface {
	// ID of the service
	ID() ExternalServiceID

	// WorkspaceID helper function, should just return workspaceID
	GetWorkspaceID() int

	// setSince takes the provided time.Time
	// and adds it to ExternalService struct. This can be used
	// to fetch just the modified data from external services.
	SetSince(*time.Time)

	// setParams takes the necessary ExternalService params
	// (for example the selected account id) as JSON
	// and adds them to ExternalService struct.
	SetParams([]byte) error

	// SetAuthData adds the provided oauth token to ExternalService struct
	SetAuthData([]byte) error

	// keyFor should provide unique key for object type
	// Example: asana:account:XXXX:projects
	KeyFor(PipeID) string

	// Accounts maps foreign account to Account models
	// https://github.com/toggl/pipes-api/blob/master/model.go#L9-L12
	Accounts() ([]*toggl.Account, error)

	// Users maps foreign users to User models
	// https://github.com/toggl/pipes-api/blob/master/model.go#L14-L19
	Users() ([]*toggl.User, error)

	// Clients maps foreign clients to Client models
	// https://github.com/toggl/pipes-api/blob/master/model.go#L21-L25
	Clients() ([]*toggl.Client, error)

	// Projects maps foreign projects to Project models
	// https://github.com/toggl/pipes-api/blob/master/model.go#L27-L36
	Projects() ([]*toggl.Project, error)

	// Tasks maps foreign tasks to Task models
	// https://github.com/toggl/pipes-api/blob/master/model.go#L38-L45
	Tasks() ([]*toggl.Task, error)

	// TodoLists maps foreign todo lists to Task models
	// https://github.com/toggl/pipes-api/blob/master/model.go#L38-45
	TodoLists() ([]*toggl.Task, error)

	// Exports time entry model to foreign service
	// should return foreign id of saved time entry
	// https://github.com/toggl/pipes-api/blob/master/model.go#L47-L61
	ExportTimeEntry(*toggl.TimeEntry) (int, error)
}

type ExternalServiceID string

const (
	BaseCamp   ExternalServiceID = "basecamp"
	FreshBooks ExternalServiceID = "freshbooks"
	TeamWeek   ExternalServiceID = "teamweek"
	Asana      ExternalServiceID = "asana"
	GitHub     ExternalServiceID = "github"
)

type PipeID string

const (
	UsersPipe       PipeID = "users"
	ClientsPipe     PipeID = "clients"
	ProjectsPipe    PipeID = "projects"
	TasksPipe       PipeID = "tasks"
	TodoPipe        PipeID = "todolists"
	TimeEntriesPipe PipeID = "time_entries"
)
