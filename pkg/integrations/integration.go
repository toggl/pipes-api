package integrations

import (
	"fmt"
	"time"

	"github.com/toggl/pipes-api/pkg/integrations/asana"
	"github.com/toggl/pipes-api/pkg/integrations/basecamp"
	"github.com/toggl/pipes-api/pkg/integrations/freshbooks"
	"github.com/toggl/pipes-api/pkg/integrations/github"
	"github.com/toggl/pipes-api/pkg/integrations/teamweek"
	"github.com/toggl/pipes-api/pkg/toggl"
)

type Integration struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	Link       string  `json:"link"`
	Image      string  `json:"image"`
	AuthURL    string  `json:"auth_url,omitempty"`
	AuthType   string  `json:"auth_type,omitempty"`
	Authorized bool    `json:"authorized"`
	Pipes      []*Pipe `json:"pipes"`
}

type (
	// ExternalService interface for external integrations
	// Example implementation: github.go
	ExternalService interface {
		// Name of the service
		Name() string

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
		KeyFor(string) string

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
)

func Create(serviceID string, workspaceID int) ExternalService {
	switch serviceID {
	case basecamp.ServiceName:
		return &basecamp.Service{WorkspaceID: workspaceID}
	case freshbooks.ServiceName:
		return &freshbooks.Service{WorkspaceID: workspaceID}
	case teamweek.ServiceName:
		return &teamweek.Service{WorkspaceID: workspaceID}
	case asana.ServiceName:
		return &asana.Service{WorkspaceID: workspaceID}
	case github.ServiceName:
		return &github.Service{WorkspaceID: workspaceID}
	default:
		panic(fmt.Sprintf("getService: Unrecognized serviceID - %s", serviceID))
	}
}

var _ ExternalService = (*basecamp.Service)(nil)
var _ ExternalService = (*freshbooks.Service)(nil)
var _ ExternalService = (*teamweek.Service)(nil)
var _ ExternalService = (*asana.Service)(nil)
var _ ExternalService = (*github.Service)(nil)
