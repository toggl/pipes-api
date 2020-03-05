package pipe

import (
	"github.com/toggl/pipes-api/pkg/integrations"
	"github.com/toggl/pipes-api/pkg/toggl"
)

//go:generate mockery -name Storage -case underscore -inpkg
type Storage interface {
	Queue

	IsDown() bool

	ClearImportFor(s integrations.ExternalService, pid integrations.PipeID) error

	LoadAccounts(s integrations.ExternalService) (*toggl.AccountsResponse, error)
	LoadPipe(workspaceID int, sid integrations.ExternalServiceID, pid integrations.PipeID) (*Pipe, error)
	LoadPipeStatus(workspaceID int, sid integrations.ExternalServiceID, pid integrations.PipeID) (*Status, error)
	LoadAuthorization(workspaceID int, sid integrations.ExternalServiceID) (*Authorization, error)
	LoadConnection(workspaceID int, key string) (*Connection, error)
	LoadReversedConnection(workspaceID int, key string) (*ReversedConnection, error)
	LoadPipes(workspaceID int) (map[string]*Pipe, error)
	LoadLastSync(p *Pipe)
	LoadPipeStatuses(workspaceID int) (map[string]*Status, error)
	LoadWorkspaceAuthorizations(workspaceID int) (map[integrations.ExternalServiceID]bool, error)

	Delete(p *Pipe, workspaceID int) error
	DeletePipeByWorkspaceIDServiceID(workspaceID int, sid integrations.ExternalServiceID) error
	DeletePipeConnections(workspaceID int, pipeConnectionKey, pipeStatusKey string) (err error)
	DeleteAuthorization(workspaceID int, externalServiceID integrations.ExternalServiceID) error

	Save(p *Pipe) error
	SaveConnection(c *Connection) error
	SavePipeStatus(p *Status) error
	SaveAuthorization(a *Authorization) error
	SaveAccounts(s integrations.ExternalService) error

	LoadObject(s integrations.ExternalService, pid integrations.PipeID) ([]byte, error)
	SaveObject(s integrations.ExternalService, pid integrations.PipeID, obj interface{}) error
}
