package cfg

type Authorization struct {
	WorkspaceID    int
	ServiceID      string
	WorkspaceToken string
	Data           []byte
}

func NewAuthorization(workspaceID int, serviceID string) *Authorization {
	return &Authorization{
		WorkspaceID: workspaceID,
		ServiceID:   serviceID,
	}
}
