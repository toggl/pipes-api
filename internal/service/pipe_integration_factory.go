package service

import (
	"fmt"

	"github.com/toggl/pipes-api/pkg/domain"
	"github.com/toggl/pipes-api/pkg/integration/asana"
	"github.com/toggl/pipes-api/pkg/integration/basecamp"
	"github.com/toggl/pipes-api/pkg/integration/freshbooks"
	"github.com/toggl/pipes-api/pkg/integration/github"
	"github.com/toggl/pipes-api/pkg/integration/togglplan"
)

func NewPipeIntegration(id domain.IntegrationID, workspaceID int) domain.PipeIntegration {
	switch id {
	case domain.BaseCamp:
		return &basecamp.Service{WorkspaceID: workspaceID}
	case domain.FreshBooks:
		return &freshbooks.Service{WorkspaceID: workspaceID}
	case domain.TogglPlan:
		return &togglplan.Service{WorkspaceID: workspaceID}
	case domain.Asana:
		return &asana.Service{WorkspaceID: workspaceID}
	case domain.GitHub:
		return &github.Service{WorkspaceID: workspaceID}
	default:
		panic(fmt.Sprintf("getService: Unrecognized integrations.IntegrationID - %s", id))
	}
}

var _ domain.PipeIntegration = (*basecamp.Service)(nil)
var _ domain.PipeIntegration = (*freshbooks.Service)(nil)
var _ domain.PipeIntegration = (*togglplan.Service)(nil)
var _ domain.PipeIntegration = (*asana.Service)(nil)
var _ domain.PipeIntegration = (*github.Service)(nil)
