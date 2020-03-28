package pipe

import (
	"time"

	goauth2 "code.google.com/p/goauth2/oauth"
	"github.com/tambet/oauthplain"

	"github.com/toggl/pipes-api/pkg/integration"
	"github.com/toggl/pipes-api/pkg/toggl"
)

//go:generate mockery -name TogglClient -case underscore -inpkg
type TogglClient interface {
	WithAuthToken(authToken string)
	GetWorkspaceIdByToken(token string) (int, error)
	PostClients(clientsPipeID integration.PipeID, clients interface{}) (*toggl.ClientsImport, error)
	PostProjects(projectsPipeID integration.PipeID, projects interface{}) (*toggl.ProjectsImport, error)
	PostTasks(tasksPipeID integration.PipeID, tasks interface{}) (*toggl.TasksImport, error)
	PostTodoLists(tasksPipeID integration.PipeID, tasks interface{}) (*toggl.TasksImport, error)
	PostUsers(usersPipeID integration.PipeID, users interface{}) (*toggl.UsersImport, error)
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

	GetPipe(workspaceID int, sid integration.ID, pid integration.PipeID) (*Pipe, error)
	CreatePipe(workspaceID int, sid integration.ID, pid integration.PipeID, params []byte) error
	UpdatePipe(workspaceID int, sid integration.ID, pid integration.PipeID, params []byte) error
	DeletePipe(workspaceID int, sid integration.ID, pid integration.PipeID) error
	RunPipe(workspaceID int, sid integration.ID, pid integration.PipeID, usersSelector []byte) error
	GetServicePipeLog(workspaceID int, sid integration.ID, pid integration.PipeID) (string, error)

	ClearIDMappings(workspaceID int, sid integration.ID, pid integration.PipeID) error // TODO: Remove (Probably dead method).

	GetServiceUsers(workspaceID int, sid integration.ID, forceImport bool) (*toggl.UsersResponse, error)

	GetServiceAccounts(workspaceID int, sid integration.ID, forceImport bool) (*toggl.AccountsResponse, error)

	GetAuthURL(sid integration.ID, accountName, callbackURL string) (string, error)
	// CreateAuthorization creates new authorization for specified workspace and service and stores it in the persistent storage.
	// workspaceToken - it is an "Toggl.Track" authorization token which is "user_name" field from BasicAuth HTTP Header. E.g.: "Authorization Bearer base64(user_name:password)".
	CreateAuthorization(workspaceID int, sid integration.ID, workspaceToken string, params AuthParams) error
	// DeleteAuthorization removes authorization for specified workspace and service from the persistent storage.
	// It also delete all pipes for given service and workspace.
	DeleteAuthorization(workspaceID int, sid integration.ID) error

	GetIntegrations(workspaceID int) ([]Integration, error)

	Ready() []error
}

//go:generate mockery -name Storage -case underscore -inpkg
type Storage interface {
	// ID Mappings (Connections)
	LoadIDMapping(workspaceID int, key string) (*IDMapping, error)
	LoadReversedIDMapping(workspaceID int, key string) (*ReversedIDMapping, error)
	SaveIDMapping(c *IDMapping) error
	DeleteIDMappings(workspaceID int, pipeConnectionKey, pipeStatusKey string) (err error)

	// Pipes

	LoadPipe(workspaceID int, sid integration.ID, pid integration.PipeID) (*Pipe, error)
	LoadPipes(workspaceID int) (map[string]*Pipe, error)
	Save(p *Pipe) error
	Delete(p *Pipe, workspaceID int) error
	DeletePipesByWorkspaceIDServiceID(workspaceID int, sid integration.ID) error
	LoadLastSync(p *Pipe)

	// Pipe statuses

	LoadPipeStatus(workspaceID int, sid integration.ID, pid integration.PipeID) (*Status, error)
	LoadPipeStatuses(workspaceID int) (map[string]*Status, error)
	SavePipeStatus(p *Status) error

	IsDown() bool
}

//go:generate mockery -name AuthorizationsStorage -case underscore -inpkg
type AuthorizationsStorage interface {
	LoadAuthorization(workspaceID int, externalServiceID integration.ID, a *Authorization) error
	LoadWorkspaceAuthorizations(workspaceID int) (map[integration.ID]bool, error)
	SaveAuthorization(a *Authorization) error
	DeleteAuthorization(workspaceID int, externalServiceID integration.ID) error
}

//go:generate mockery -name IntegrationsStorage -case underscore -inpkg
type IntegrationsStorage interface {
	LoadIntegrations() ([]*Integration, error)
	LoadAuthorizationType(serviceID integration.ID) (string, error)
	SaveAuthorizationType(serviceID integration.ID, authType string) error

	IsValidPipe(pipeID integration.PipeID) bool
	IsValidService(serviceID integration.ID) bool
}

//go:generate mockery -name ImportsStorage -case underscore -inpkg
type ImportsStorage interface {
	// Imports
	LoadAccountsFor(s integration.Integration) (*toggl.AccountsResponse, error)
	SaveAccountsFor(s integration.Integration, res toggl.AccountsResponse) error
	DeleteAccountsFor(s integration.Integration) error

	LoadUsersFor(s integration.Integration) (*toggl.UsersResponse, error)
	SaveUsersFor(s integration.Integration, res toggl.UsersResponse) error
	DeleteUsersFor(s integration.Integration) error

	LoadClientsFor(s integration.Integration) (*toggl.ClientsResponse, error)
	SaveClientsFor(s integration.Integration, res toggl.ClientsResponse) error

	LoadProjectsFor(s integration.Integration) (*toggl.ProjectsResponse, error)
	SaveProjectsFor(s integration.Integration, res toggl.ProjectsResponse) error

	LoadTasksFor(s integration.Integration) (*toggl.TasksResponse, error)
	SaveTasksFor(s integration.Integration, res toggl.TasksResponse) error

	LoadTodoListsFor(s integration.Integration) (*toggl.TasksResponse, error)
	SaveTodoListsFor(s integration.Integration, res toggl.TasksResponse) error
}

//go:generate mockery -name OAuthProvider -case underscore -inpkg
type OAuthProvider interface {
	OAuth2URL(integration.ID) string
	OAuth1Configs(integration.ID) (*oauthplain.Config, bool)
	OAuth1Exchange(sid integration.ID, accountName, oAuthToken, oAuthVerifier string) (*oauthplain.Token, error)
	OAuth2Exchange(sid integration.ID, code string) (*goauth2.Token, error)
	OAuth2Configs(integration.ID) (*goauth2.Config, bool)
	OAuth2Refresh(*goauth2.Config, *goauth2.Token) error
}
