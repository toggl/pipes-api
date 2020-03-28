package domain

import (
	"fmt"

	"github.com/toggl/pipes-api/pkg/integration"
	"github.com/toggl/pipes-api/pkg/integration/asana"
	"github.com/toggl/pipes-api/pkg/integration/basecamp"
	"github.com/toggl/pipes-api/pkg/integration/freshbooks"
	"github.com/toggl/pipes-api/pkg/integration/github"
	"github.com/toggl/pipes-api/pkg/integration/teamweek"
)

func NewExternalService(id integration.ID, workspaceID int) integration.Integration {
	switch id {
	case integration.BaseCamp:
		return &basecamp.Service{WorkspaceID: workspaceID}
	case integration.FreshBooks:
		return &freshbooks.Service{WorkspaceID: workspaceID}
	case integration.TeamWeek:
		return &teamweek.Service{WorkspaceID: workspaceID}
	case integration.Asana:
		return &asana.Service{WorkspaceID: workspaceID}
	case integration.GitHub:
		return &github.Service{WorkspaceID: workspaceID}
	default:
		panic(fmt.Sprintf("getService: Unrecognized integrations.ID - %s", id))
	}
}

var _ integration.Integration = (*basecamp.Service)(nil)
var _ integration.Integration = (*freshbooks.Service)(nil)
var _ integration.Integration = (*teamweek.Service)(nil)
var _ integration.Integration = (*asana.Service)(nil)
var _ integration.Integration = (*github.Service)(nil)
