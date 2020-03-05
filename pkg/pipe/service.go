package pipe

import (
	"github.com/toggl/pipes-api/pkg/integrations"
	"github.com/toggl/pipes-api/pkg/toggl"
)

//go:generate mockery -name Service -case underscore -output ./mocks
type Service interface {
	QueueRunner

	GetIntegrationPipe(workspaceID int, serviceID integrations.ExternalServiceID, pipeID integrations.PipeID) (*Pipe, error)
	CreatePipe(workspaceID int, serviceID integrations.ExternalServiceID, pipeID integrations.PipeID, params []byte) error
	UpdatePipe(workspaceID int, serviceID integrations.ExternalServiceID, pipeID integrations.PipeID, params []byte) error
	DeletePipe(workspaceID int, serviceID integrations.ExternalServiceID, pipeID integrations.PipeID) error
	GetServicePipeLog(workspaceID int, serviceID integrations.ExternalServiceID, pipeID integrations.PipeID) (string, error)
	ClearPipeConnections(workspaceID int, serviceID integrations.ExternalServiceID, pipeID integrations.PipeID) error
	RunPipe(workspaceID int, serviceID integrations.ExternalServiceID, pipeID integrations.PipeID, params []byte) error
	GetServiceUsers(workspaceID int, serviceID integrations.ExternalServiceID, forceImport bool) (*toggl.UsersResponse, error)
	GetServiceAccounts(workspaceID int, serviceID integrations.ExternalServiceID, forceImport bool) (*toggl.AccountsResponse, error)
	GetAuthURL(serviceID integrations.ExternalServiceID, accountName, callbackURL string) (string, error)
	CreateAuthorization(workspaceID int, serviceID integrations.ExternalServiceID, currentWorkspaceToken string, oAuthRawData []byte) error
	DeleteAuthorization(workspaceID int, serviceID integrations.ExternalServiceID) error
	WorkspaceIntegrations(workspaceID int) ([]Integration, error)
	Ready() []error
	AvailablePipeType(pipeID integrations.PipeID) bool
	AvailableServiceType(serviceID integrations.ExternalServiceID) bool
}
