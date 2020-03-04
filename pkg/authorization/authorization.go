package authorization

import (
	"encoding/json"

	"code.google.com/p/goauth2/oauth"

	"github.com/toggl/pipes-api/pkg/integrations"
)

const (
	TypeOauth2 = "oauth2"
	TypeOauth1 = "oauth1"
)

type Authorization struct {
	WorkspaceID    int
	ServiceID      integrations.ExternalServiceID
	WorkspaceToken string
	Data           []byte
}

func New(workspaceID int, id integrations.ExternalServiceID) *Authorization {
	return &Authorization{
		WorkspaceID: workspaceID,
		ServiceID:   id,
		Data:        []byte("{}"),
	}
}

func (a *Authorization) SetOauth2Token(t oauth.Token) error {
	b, err := json.Marshal(t)
	if err != nil {
		return err
	}
	a.Data = b
	return nil
}
