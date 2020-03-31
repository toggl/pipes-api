package domain

//go:generate mockery -name PipeService -case underscore -outpkg mocks
type PipeService interface {
	GetPipe(workspaceID int, sid IntegrationID, pid PipeID) (*Pipe, error)
	CreatePipe(workspaceID int, sid IntegrationID, pid PipeID, params []byte) error
	UpdatePipe(workspaceID int, sid IntegrationID, pid PipeID, params []byte) error
	DeletePipe(workspaceID int, sid IntegrationID, pid PipeID) error
}

//go:generate mockery -name PipeSyncService -case underscore -outpkg mocks
type PipeSyncService interface {
	GetServicePipeLog(workspaceID int, sid IntegrationID, pid PipeID) (string, error)
	GetServiceUsers(workspaceID int, sid IntegrationID, forceImport bool) (*UsersResponse, error)
	GetServiceAccounts(workspaceID int, sid IntegrationID, forceImport bool) (*AccountsResponse, error)
	GetIntegrations(workspaceID int) ([]Integration, error)
	Synchronize(p *Pipe)

	// Deprecated: TODO: Remove dead method. It's used only in h4xx0rz(old Backoffice) https://github.com/toggl/support/blob/master/app/controllers/workspaces_controller.rb#L145
	ClearIDMappings(workspaceID int, sid IntegrationID, pid PipeID) error
}

//go:generate mockery -name AuthorizationService -case underscore -outpkg mocks
type AuthorizationService interface {
	GetAuthURL(sid IntegrationID, accountName, callbackURL string) (string, error)
	// CreateAuthorization creates new authorization for specified workspace and service and stores it in the persistent storage.
	// workspaceToken - it is an "Toggl.Track" authorization token which is "user_name" field from BasicAuth HTTP Header. E.g.: "Authorization Bearer base64(user_name:password)".
	CreateAuthorization(workspaceID int, sid IntegrationID, workspaceToken string, params AuthParams) error
	// DeleteAuthorization removes authorization for specified workspace and service from the persistent storage.
	// It also delete all pipes for given service and workspace.
	DeleteAuthorization(workspaceID int, sid IntegrationID) error
}

//go:generate mockery -name HealthCheckService -case underscore -outpkg mocks
type HealthCheckService interface {
	Ready() []error
}
