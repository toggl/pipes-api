package pipe

import (
	"github.com/toggl/pipes-api/pkg/integrations"
	"github.com/toggl/pipes-api/pkg/toggl"
)

//go:generate mockery -name Service -case underscore -inpkg
type Service interface {
	QueueRunner

	GetPipe(workspaceID int, sid integrations.ExternalServiceID, pid integrations.PipeID) (*Pipe, error)
	CreatePipe(workspaceID int, sid integrations.ExternalServiceID, pid integrations.PipeID, params []byte) error
	UpdatePipe(workspaceID int, sid integrations.ExternalServiceID, pid integrations.PipeID, params []byte) error
	DeletePipe(workspaceID int, sid integrations.ExternalServiceID, pid integrations.PipeID) error
	GetServicePipeLog(workspaceID int, sid integrations.ExternalServiceID, pid integrations.PipeID) (string, error)
	ClearPipeConnections(workspaceID int, sid integrations.ExternalServiceID, pid integrations.PipeID) error
	RunPipe(workspaceID int, sid integrations.ExternalServiceID, pid integrations.PipeID, payload []byte) error
	GetServiceUsers(workspaceID int, sid integrations.ExternalServiceID, forceImport bool) (*toggl.UsersResponse, error)
	GetServiceAccounts(workspaceID int, sid integrations.ExternalServiceID, forceImport bool) (*toggl.AccountsResponse, error)
	GetAuthURL(sid integrations.ExternalServiceID, accountName, callbackURL string) (string, error)
	CreateAuthorization(workspaceID int, sid integrations.ExternalServiceID, currentWorkspaceToken string, oAuthRawData []byte) error
	DeleteAuthorization(workspaceID int, sid integrations.ExternalServiceID) error
	WorkspaceIntegrations(workspaceID int) ([]Integration, error)
	Ready() []error
	AvailablePipeType(pid integrations.PipeID) bool
	AvailableServiceType(sid integrations.ExternalServiceID) bool
}
