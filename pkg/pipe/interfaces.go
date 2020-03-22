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
	RunPipe(workspaceID int, sid integrations.ExternalServiceID, pid integrations.PipeID, usersSelector []byte) error
	GetServicePipeLog(workspaceID int, sid integrations.ExternalServiceID, pid integrations.PipeID) (string, error)

	ClearIDMappings(workspaceID int, sid integrations.ExternalServiceID, pid integrations.PipeID) error // TODO: Remove (Probably dead method).

	GetServiceUsers(workspaceID int, sid integrations.ExternalServiceID, forceImport bool) (*toggl.UsersResponse, error)

	GetServiceAccounts(workspaceID int, sid integrations.ExternalServiceID, forceImport bool) (*toggl.AccountsResponse, error)

	GetAuthURL(sid integrations.ExternalServiceID, accountName, callbackURL string) (string, error)
	// CreateAuthorization creates new authorization for specified workspace and service and stores it in the persistent storage.
	// workspaceToken - it is an "Toggl.Track" authorization token which is "user_name" field from BasicAuth HTTP Header. E.g.: "Authorization Bearer base64(user_name:password)".
	CreateAuthorization(workspaceID int, sid integrations.ExternalServiceID, workspaceToken string, params AuthParams) error
	// DeleteAuthorization removes authorization for specified workspace and service from the persistent storage.
	// It also delete all pipes for given service and workspace.
	DeleteAuthorization(workspaceID int, sid integrations.ExternalServiceID) error

	GetIntegrations(workspaceID int) ([]Integration, error)

	Ready() []error
}

//go:generate mockery -name Storage -case underscore -inpkg
type Storage interface {
	// Authorizations

	LoadAuthorization(workspaceID int, sid integrations.ExternalServiceID) (*Authorization, error)
	LoadWorkspaceAuthorizations(workspaceID int) (map[integrations.ExternalServiceID]bool, error)
	SaveAuthorization(a *Authorization) error
	DeleteAuthorization(workspaceID int, externalServiceID integrations.ExternalServiceID) error

	// ID Mappings (Connections)

	LoadIDMapping(workspaceID int, key string) (*IDMapping, error)
	LoadReversedIDMapping(workspaceID int, key string) (*ReversedIDMapping, error)
	SaveIDMapping(c *IDMapping) error
	DeleteIDMappings(workspaceID int, pipeConnectionKey, pipeStatusKey string) (err error)

	// Pipes

	LoadPipe(workspaceID int, sid integrations.ExternalServiceID, pid integrations.PipeID) (*Pipe, error)
	LoadPipes(workspaceID int) (map[string]*Pipe, error)
	Save(p *Pipe) error
	Delete(p *Pipe, workspaceID int) error
	DeletePipesByWorkspaceIDServiceID(workspaceID int, sid integrations.ExternalServiceID) error
	LoadLastSync(p *Pipe)

	// Pipe statuses

	LoadPipeStatus(workspaceID int, sid integrations.ExternalServiceID, pid integrations.PipeID) (*Status, error)
	LoadPipeStatuses(workspaceID int) (map[string]*Status, error)
	SavePipeStatus(p *Status) error

	IsDown() bool
}

//go:generate mockery -name IntegrationsStorage -case underscore -inpkg
type IntegrationsStorage interface {
	LoadIntegrations() ([]*Integration, error)
	LoadAuthorizationType(serviceID integrations.ExternalServiceID) (string, error)
	SaveAuthorizationType(serviceID integrations.ExternalServiceID, authType string) error

	IsValidPipe(pipeID integrations.PipeID) bool
	IsValidService(serviceID integrations.ExternalServiceID) bool
}

//go:generate mockery -name ImportsStorage -case underscore -inpkg
type ImportsStorage interface {
	// Imports
	LoadAccountsFor(s integrations.ExternalService) (*toggl.AccountsResponse, error)
	SaveAccountsFor(s integrations.ExternalService, res toggl.AccountsResponse) error
	DeleteAccountsFor(s integrations.ExternalService) error

	LoadUsersFor(s integrations.ExternalService) (*toggl.UsersResponse, error)
	SaveUsersFor(s integrations.ExternalService, res toggl.UsersResponse) error
	DeleteUsersFor(s integrations.ExternalService) error

	LoadClientsFor(s integrations.ExternalService) (*toggl.ClientsResponse, error)
	SaveClientsFor(s integrations.ExternalService, res toggl.ClientsResponse) error

	LoadProjectsFor(s integrations.ExternalService) (*toggl.ProjectsResponse, error)
	SaveProjectsFor(s integrations.ExternalService, res toggl.ProjectsResponse) error

	LoadTasksFor(s integrations.ExternalService) (*toggl.TasksResponse, error)
	SaveTasksFor(s integrations.ExternalService, res toggl.TasksResponse) error

	LoadTodoListsFor(s integrations.ExternalService) (*toggl.TasksResponse, error)
	SaveTodoListsFor(s integrations.ExternalService, res toggl.TasksResponse) error
}
