package integration

import (
	"fmt"

	"github.com/toggl/pipes-api/pkg/domain"
)

func NewPipeIntegration(id domain.IntegrationID, workspaceID int) domain.PipeIntegration {
	switch id {
	case domain.BaseCamp:
		return &BaseCampPipeIntegration{WorkspaceID: workspaceID}
	case domain.FreshBooks:
		return &FreshBooksPipeIntegration{WorkspaceID: workspaceID}
	case domain.TogglPlan:
		return &TogglPlanPipeIntegration{WorkspaceID: workspaceID}
	case domain.Asana:
		return &AsanaPipeIntegration{WorkspaceID: workspaceID}
	case domain.GitHub:
		return &GitHubPipeIntegration{WorkspaceID: workspaceID}
	default:
		panic(fmt.Sprintf("getService: Unrecognized integrations.IntegrationID - %s", id))
	}
}

var _ domain.PipeIntegration = (*BaseCampPipeIntegration)(nil)
var _ domain.PipeIntegration = (*FreshBooksPipeIntegration)(nil)
var _ domain.PipeIntegration = (*TogglPlanPipeIntegration)(nil)
var _ domain.PipeIntegration = (*AsanaPipeIntegration)(nil)
var _ domain.PipeIntegration = (*GitHubPipeIntegration)(nil)
