package environment

type AuthorizationConfig struct {
	WorkspaceID    int
	ServiceID      string
	WorkspaceToken string
	Data           []byte
}

func NewAuthorization(workspaceID int, serviceID string) *AuthorizationConfig {
	return &AuthorizationConfig{
		WorkspaceID: workspaceID,
		ServiceID:   serviceID,
	}
}
