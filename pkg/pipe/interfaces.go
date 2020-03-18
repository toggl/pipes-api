package pipe

import (
	"time"

	"github.com/toggl/pipes-api/pkg/integrations"
	"github.com/toggl/pipes-api/pkg/toggl"
)

//go:generate mockery -name TogglClient -case underscore -inpkg
type TogglClient interface {
	WithAuthToken(authToken string)
	GetWorkspaceIdByToken(token string) (int, error)
	PostClients(clientsPipeID integrations.PipeID, clients interface{}) (*toggl.ClientsImport, error)
	PostProjects(projectsPipeID integrations.PipeID, projects interface{}) (*toggl.ProjectsImport, error)
	PostTasks(tasksPipeID integrations.PipeID, tasks interface{}) (*toggl.TasksImport, error)
	PostTodoLists(tasksPipeID integrations.PipeID, tasks interface{}) (*toggl.TasksImport, error)
	PostUsers(usersPipeID integrations.PipeID, users interface{}) (*toggl.UsersImport, error)
	GetTimeEntries(lastSync time.Time, userIDs, projectsIDs []int) ([]toggl.TimeEntry, error)
	AdjustRequestSize(tasks []*toggl.Task, split int) ([]*toggl.TaskRequest, error)
	Ping() error
}

//go:generate mockery -name Runner -case underscore -inpkg
type Runner interface {
	Run(*Pipe)
}

//go:generate mockery -name Queue -case underscore -inpkg
type Queue interface {
	QueueAutomaticPipes() error
	GetPipesFromQueue() ([]*Pipe, error)
	SetQueuedPipeSynced(*Pipe) error
	QueuePipeAsFirst(*Pipe) error
}

//go:generate mockery -name Service -case underscore -inpkg
type Service interface {
	Runner

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
