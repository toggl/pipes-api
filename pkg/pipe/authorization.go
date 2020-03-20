package pipe

import (
	"encoding/json"

	goauth2 "code.google.com/p/goauth2/oauth"
	"github.com/tambet/oauthplain"

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
	// Data can store 2 different structures encoded to JSON depends on Authorization type.
	// For oAuth v1 it will store "*oauthplain.Token" and for oAuth v2 it will store "*goauth2.Token".
	Data []byte
}

func NewAuthorization(workspaceID int, id integrations.ExternalServiceID, workspaceToken string) *Authorization {
	return &Authorization{
		WorkspaceID:    workspaceID,
		ServiceID:      id,
		WorkspaceToken: workspaceToken,
		Data:           []byte("{}"),
	}
}

func (a *Authorization) SetOAuth2Token(t *goauth2.Token) error {
	b, err := json.Marshal(t)
	if err != nil {
		return err
	}
	a.Data = b
	return nil
}

func (a *Authorization) SetOAuth1Token(t *oauthplain.Token) error {
	b, err := json.Marshal(t)
	if err != nil {
		return err
	}
	a.Data = b
	return nil
}
