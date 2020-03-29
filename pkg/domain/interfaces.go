package domain

import (
	"time"

	goauth2 "code.google.com/p/goauth2/oauth"
	"github.com/tambet/oauthplain"
)

//go:generate mockery -name TogglClient -case underscore -outpkg mocks
type TogglClient interface {
	WithAuthToken(authToken string)
	GetWorkspaceIdByToken(token string) (int, error)
	PostClients(clientsPipeID PipeID, clients interface{}) (*ClientsImport, error)
	PostProjects(projectsPipeID PipeID, projects interface{}) (*ProjectsImport, error)
	PostTasks(tasksPipeID PipeID, tasks interface{}) (*TasksImport, error)
	PostTodoLists(tasksPipeID PipeID, tasks interface{}) (*TasksImport, error)
	PostUsers(usersPipeID PipeID, users interface{}) (*UsersImport, error)
	GetTimeEntries(lastSync time.Time, userIDs, projectsIDs []int) ([]TimeEntry, error)
	AdjustRequestSize(tasks []*Task, split int) ([]*TaskRequest, error)
	Ping() error
}

//go:generate mockery -name Queue -case underscore -outpkg mocks
type Queue interface {
	ScheduleAutomaticPipesSynchronization() error
	LoadScheduledPipes() ([]*Pipe, error)
	MarkPipeSynchronized(*Pipe) error
	SchedulePipeSynchronization(workspaceID int, serviceID ID, pipeID PipeID, usersSelector UserParams) error
}

//go:generate mockery -name PipeService -case underscore -outpkg mocks
type PipeService interface {
	GetPipe(workspaceID int, sid ID, pid PipeID) (*Pipe, error)
	CreatePipe(workspaceID int, sid ID, pid PipeID, params []byte) error
	UpdatePipe(workspaceID int, sid ID, pid PipeID, params []byte) error
	DeletePipe(workspaceID int, sid ID, pid PipeID) error
	GetServicePipeLog(workspaceID int, sid ID, pid PipeID) (string, error)

	// Deprecated: TODO: Remove dead method. It's used only in h4xx0rz(old Backoffice) https://github.com/toggl/support/blob/master/app/controllers/workspaces_controller.rb#L145
	ClearIDMappings(workspaceID int, sid ID, pid PipeID) error

	GetServiceUsers(workspaceID int, sid ID, forceImport bool) (*UsersResponse, error)

	GetServiceAccounts(workspaceID int, sid ID, forceImport bool) (*AccountsResponse, error)

	GetAuthURL(sid ID, accountName, callbackURL string) (string, error)
	// CreateAuthorization creates new authorization for specified workspace and service and stores it in the persistent storage.
	// workspaceToken - it is an "Toggl.Track" authorization token which is "user_name" field from BasicAuth HTTP Header. E.g.: "Authorization Bearer base64(user_name:password)".
	CreateAuthorization(workspaceID int, sid ID, workspaceToken string, params AuthParams) error
	// DeleteAuthorization removes authorization for specified workspace and service from the persistent storage.
	// It also delete all pipes for given service and workspace.
	DeleteAuthorization(workspaceID int, sid ID) error

	GetIntegrations(workspaceID int) ([]Integration, error)

	Synchronize(p *Pipe)

	Ready() []error
}

//go:generate mockery -name PipesStorage -case underscore -outpkg mocks
type PipesStorage interface {
	// Pipes
	Load(p *Pipe) error
	LoadAll(workspaceID int) (map[string]*Pipe, error)
	Save(p *Pipe) error
	Delete(p *Pipe, workspaceID int) error
	DeleteByWorkspaceIDServiceID(workspaceID int, sid ID) error
	LoadLastSyncFor(p *Pipe)

	// Pipe Statuses
	LoadStatus(workspaceID int, sid ID, pid PipeID) (*Status, error)
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
	Load(workspaceID int, externalServiceID ID, a *Authorization) error
	LoadWorkspaceAuthorizations(workspaceID int) (map[ID]bool, error)
	Save(a *Authorization) error
	Delete(workspaceID int, externalServiceID ID) error
}

//go:generate mockery -name IntegrationsStorage -case underscore -outpkg mocks
type IntegrationsStorage interface {
	LoadIntegrations() ([]*Integration, error)
	LoadAuthorizationType(serviceID ID) (string, error)
	SaveAuthorizationType(serviceID ID, authType string) error

	IsValidPipe(pipeID PipeID) bool
	IsValidService(serviceID ID) bool
}

//go:generate mockery -name ImportsStorage -case underscore -outpkg mocks
type ImportsStorage interface {
	// Imports
	LoadAccountsFor(s PipeIntegration) (*AccountsResponse, error)
	SaveAccountsFor(s PipeIntegration, res AccountsResponse) error
	DeleteAccountsFor(s PipeIntegration) error

	LoadUsersFor(s PipeIntegration) (*UsersResponse, error)
	SaveUsersFor(s PipeIntegration, res UsersResponse) error
	DeleteUsersFor(s PipeIntegration) error

	LoadClientsFor(s PipeIntegration) (*ClientsResponse, error)
	SaveClientsFor(s PipeIntegration, res ClientsResponse) error

	LoadProjectsFor(s PipeIntegration) (*ProjectsResponse, error)
	SaveProjectsFor(s PipeIntegration, res ProjectsResponse) error

	LoadTasksFor(s PipeIntegration) (*TasksResponse, error)
	SaveTasksFor(s PipeIntegration, res TasksResponse) error

	LoadTodoListsFor(s PipeIntegration) (*TasksResponse, error)
	SaveTodoListsFor(s PipeIntegration, res TasksResponse) error
}

//go:generate mockery -name OAuthProvider -case underscore -outpkg mocks
type OAuthProvider interface {
	OAuth2URL(ID) string
	OAuth1Configs(ID) (*oauthplain.Config, bool)
	OAuth1Exchange(sid ID, accountName, oAuthToken, oAuthVerifier string) (*oauthplain.Token, error)
	OAuth2Exchange(sid ID, code string) (*goauth2.Token, error)
	OAuth2Configs(ID) (*goauth2.Config, bool)
	OAuth2Refresh(*goauth2.Config, *goauth2.Token) error
}
