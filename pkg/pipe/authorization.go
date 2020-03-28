package pipe

import (
	"encoding/json"
	"errors"
	"fmt"

	goauth2 "code.google.com/p/goauth2/oauth"
	"github.com/tambet/oauthplain"

	"github.com/toggl/pipes-api/pkg/integration"
)

const (
	TypeOauth2 = "oauth2"
	TypeOauth1 = "oauth1"
)

type Authorization struct {
	WorkspaceID    int
	ServiceID      integration.ID
	WorkspaceToken string
	// Data can store 2 different structures encoded to JSON depends on Authorization type.
	// For oAuth v1 it will store "*oauthplain.Token" and for oAuth v2 it will store "*goauth2.Token".
	Data []byte

	IntegrationsStorage
	AuthorizationsStorage
	OAuthProvider
}

type AuthorizationFactory struct {
	IntegrationsStorage
	AuthorizationsStorage
	OAuthProvider
}

func (f *AuthorizationFactory) Create(workspaceID int, id integration.ID) *Authorization {

	if f.IntegrationsStorage == nil {
		panic("AuthorizationFactory.IntegrationsStorage should not be nil")
	}

	if f.AuthorizationsStorage == nil {
		panic("AuthorizationFactory.Storage should not be nil")
	}

	if f.OAuthProvider == nil {
		panic("AuthorizationFactory.OAuthProvider should not be nil")
	}

	return &Authorization{
		WorkspaceID: workspaceID,
		ServiceID:   id,
		Data:        []byte("{}"),

		IntegrationsStorage:   f.IntegrationsStorage,
		AuthorizationsStorage: f.AuthorizationsStorage,
		OAuthProvider:         f.OAuthProvider,
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

func (a *Authorization) Refresh() error {
	authType, err := a.IntegrationsStorage.LoadAuthorizationType(a.ServiceID)
	if err != nil {
		return err
	}
	if authType != TypeOauth2 {
		return nil
	}
	var token goauth2.Token
	if err := json.Unmarshal(a.Data, &token); err != nil {
		return err
	}
	if !token.Expired() {
		return nil
	}
	config, res := a.OAuthProvider.OAuth2Configs(a.ServiceID)
	if !res {
		return errors.New("service OAuth config not found")
	}
	if err := a.OAuthProvider.OAuth2Refresh(config, &token); err != nil {
		return fmt.Errorf("unable to refresh oAuth2 token, reason: %w", err)
	}
	if err := a.SetOAuth2Token(&token); err != nil {
		return err
	}
	if err := a.AuthorizationsStorage.Save(a); err != nil {
		return err
	}
	return nil
}
