package domain

import (
	"time"

	goauth2 "code.google.com/p/goauth2/oauth"
	"github.com/tambet/oauthplain"

	"github.com/toggl/pipes-api/pkg/integration"
	"github.com/toggl/pipes-api/pkg/toggl"
)

//go:generate mockery -name TogglClient -case underscore -outpkg mocks
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

//go:generate mockery -name Queue -case underscore -outpkg mocks
type Queue interface {
	ScheduleAutomaticPipesSynchronization() error
	LoadScheduledPipes() ([]*Pipe, error)
	MarkPipeSynchronized(*Pipe) error
	SchedulePipeSynchronization(*Pipe) error
}

//go:generate mockery -name PipeService -case underscore -outpkg mocks
type PipeService interface {
	GetPipe(workspaceID int, sid integration.ID, pid integration.PipeID) (*Pipe, error)
	CreatePipe(workspaceID int, sid integration.ID, pid integration.PipeID, params []byte) error
	UpdatePipe(workspaceID int, sid integration.ID, pid integration.PipeID, params []byte) error
	DeletePipe(workspaceID int, sid integration.ID, pid integration.PipeID) error
	RunPipe(workspaceID int, sid integration.ID, pid integration.PipeID, usersSelector UserParams) error
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

//go:generate mockery -name PipesStorage -case underscore -outpkg mocks
type PipesStorage interface {
	// Pipes
	Load(p *Pipe) error
	LoadAll(workspaceID int) (map[string]*Pipe, error)
	Save(p *Pipe) error
	Delete(p *Pipe, workspaceID int) error
	DeleteByWorkspaceIDServiceID(workspaceID int, sid integration.ID) error
	LoadLastSyncFor(p *Pipe)

	// Pipe Statuses
	LoadStatus(workspaceID int, sid integration.ID, pid integration.PipeID) (*Status, error)
	LoadAllStatuses(workspaceID int) (map[string]*Status, error)
	SaveStatus(p *Status) error

	IsDown() bool
}

//go:generate mockery -name IDMappingsStorage -case underscore -outpkg mocks
type IDMappingsStorage interface {
	Load(workspaceID int, key string) (*IDMapping, error)
	LoadReversed(workspaceID int, key string) (*ReversedIDMapping, error)
	Save(c *IDMapping) error
	Delete(workspaceID int, pipeConnectionKey, pipeStatusKey string) (err error)
}

//go:generate mockery -name AuthorizationsStorage -case underscore -outpkg mocks
type AuthorizationsStorage interface {
	Load(workspaceID int, externalServiceID integration.ID, a *Authorization) error
	LoadWorkspaceAuthorizations(workspaceID int) (map[integration.ID]bool, error)
	Save(a *Authorization) error
	Delete(workspaceID int, externalServiceID integration.ID) error
}

//go:generate mockery -name IntegrationsStorage -case underscore -outpkg mocks
type IntegrationsStorage interface {
	LoadIntegrations() ([]*Integration, error)
	LoadAuthorizationType(serviceID integration.ID) (string, error)
	SaveAuthorizationType(serviceID integration.ID, authType string) error

	IsValidPipe(pipeID integration.PipeID) bool
	IsValidService(serviceID integration.ID) bool
}

//go:generate mockery -name ImportsStorage -case underscore -outpkg mocks
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

//go:generate mockery -name OAuthProvider -case underscore -outpkg mocks
type OAuthProvider interface {
	OAuth2URL(integration.ID) string
	OAuth1Configs(integration.ID) (*oauthplain.Config, bool)
	OAuth1Exchange(sid integration.ID, accountName, oAuthToken, oAuthVerifier string) (*oauthplain.Token, error)
	OAuth2Exchange(sid integration.ID, code string) (*goauth2.Token, error)
	OAuth2Configs(integration.ID) (*goauth2.Config, bool)
	OAuth2Refresh(*goauth2.Config, *goauth2.Token) error
}
