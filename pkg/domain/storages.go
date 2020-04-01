package domain

//go:generate mockery -name PipesStorage -case underscore -outpkg mocks
type PipesStorage interface {
	// Pipes
	Load(p *Pipe) error
	LoadAll(workspaceID int) (map[string]*Pipe, error)
	Save(p *Pipe) error
	Delete(p *Pipe, workspaceID int) error
	DeleteByWorkspaceIDServiceID(workspaceID int, sid IntegrationID) error
	LoadLastSyncFor(p *Pipe)

	// Pipe Statuses
	LoadStatus(workspaceID int, sid IntegrationID, pid PipeID) (*Status, error)
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
	Load(workspaceID int, externalServiceID IntegrationID, a *Authorization) error
	LoadWorkspaceAuthorizations(workspaceID int) (map[IntegrationID]bool, error)
	Save(a *Authorization) error
	Delete(workspaceID int, externalServiceID IntegrationID) error
}

//go:generate mockery -name IntegrationsStorage -case underscore -outpkg mocks
type IntegrationsStorage interface {
	LoadIntegrations() ([]*Integration, error)
	LoadAuthorizationType(serviceID IntegrationID) (string, error)
	SaveAuthorizationType(serviceID IntegrationID, authType string) error

	IsValidPipe(pipeID PipeID) bool
	IsValidService(serviceID IntegrationID) bool
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
