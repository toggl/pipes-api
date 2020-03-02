package authorization

const (
	TypeOauth2 = "oauth2"
	TypeOauth1 = "oauth1"
)

type Authorization struct {
	WorkspaceID    int
	ServiceID      string
	WorkspaceToken string
	Data           []byte
}

func New(workspaceID int, serviceID string) *Authorization {
	return &Authorization{
		WorkspaceID: workspaceID,
		ServiceID:   serviceID,
	}
}
