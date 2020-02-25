package environment

import (
	"fmt"

	"github.com/toggl/pipes-api/pkg/integrations"
	"github.com/toggl/pipes-api/pkg/integrations/asana"
	"github.com/toggl/pipes-api/pkg/integrations/basecamp"
	"github.com/toggl/pipes-api/pkg/integrations/freshbooks"
	"github.com/toggl/pipes-api/pkg/integrations/github"
	"github.com/toggl/pipes-api/pkg/integrations/mock"
	"github.com/toggl/pipes-api/pkg/integrations/teamweek"
)

type IntegrationConfig struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	Link       string        `json:"link"`
	Image      string        `json:"image"`
	AuthURL    string        `json:"auth_url,omitempty"`
	AuthType   string        `json:"auth_type,omitempty"`
	Authorized bool          `json:"authorized"`
	Pipes      []*PipeConfig `json:"pipes"`
}

func Create(serviceID string, workspaceID int) integrations.Integration {
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
	case mock.ServiceName:
		return &mock.Service{WorkspaceID: workspaceID}
	default:
		panic(fmt.Sprintf("getService: Unrecognized serviceID - %s", serviceID))
	}
}
