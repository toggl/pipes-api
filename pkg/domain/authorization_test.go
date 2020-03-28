package domain_test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	goauth2 "code.google.com/p/goauth2/oauth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/tambet/oauthplain"

	"github.com/toggl/pipes-api/pkg/config"
	"github.com/toggl/pipes-api/pkg/domain"
	"github.com/toggl/pipes-api/pkg/domain/mocks"
	"github.com/toggl/pipes-api/pkg/integration"
)

func TestNewAuthorization(t *testing.T) {
	af := domain.AuthorizationFactory{
		IntegrationsStorage:   &mocks.IntegrationsStorage{},
		AuthorizationsStorage: &mocks.AuthorizationsStorage{},
		OAuthProvider:         &mocks.OAuthProvider{},
	}
	a := af.Create(1, integration.GitHub)
	assert.Equal(t, 1, a.WorkspaceID)
	assert.Equal(t, integration.GitHub, a.ServiceID)
	assert.NotNil(t, a.Data)
}

func TestSetOauth2Token(t *testing.T) {
	token := goauth2.Token{
		AccessToken:  "test",
		RefreshToken: "test",
		Expiry:       time.Time{},
		Extra:        nil,
	}

	af := domain.AuthorizationFactory{
		IntegrationsStorage:   &mocks.IntegrationsStorage{},
		AuthorizationsStorage: &mocks.AuthorizationsStorage{},
		OAuthProvider:         &mocks.OAuthProvider{},
	}

	a := af.Create(1, "test")
	err := a.SetOAuth2Token(&token)
	assert.NoError(t, err)

	assert.Equal(t, `{"AccessToken":"test","RefreshToken":"test","Expiry":"0001-01-01T00:00:00Z","Extra":null}`, string(a.Data))
}

func TestSetOauth1Token(t *testing.T) {
	token := oauthplain.Token{
		ConsumerKey:      "test",
		ConsumerSecret:   "test",
		OAuthToken:       "test",
		OAuthTokenSecret: "test",
		OAuthVerifier:    "test",
		AuthorizeUrl:     "test",
		Extra:            nil,
	}

	af := domain.AuthorizationFactory{
		IntegrationsStorage:   &mocks.IntegrationsStorage{},
		AuthorizationsStorage: &mocks.AuthorizationsStorage{},
		OAuthProvider:         &mocks.OAuthProvider{},
	}

	a := af.Create(1, "test")
	err := a.SetOAuth1Token(&token)
	assert.NoError(t, err)

	assert.Equal(t, `{"ConsumerKey":"test","ConsumerSecret":"test","OAuthToken":"test","OAuthTokenSecret":"test","OAuthVerifier":"test","AuthorizeUrl":"test","Extra":null}`, string(a.Data))
}

func TestRefresh_Load_Ok(t *testing.T) {

	flags := config.Flags{}
	config.ParseFlags(&flags, os.Args)

	as := &mocks.AuthorizationsStorage{}
	as.On("Load", 1, integration.GitHub, mock.Anything).Return(nil)
	as.On("Save", mock.Anything).Return(nil)

	is := &mocks.IntegrationsStorage{}
	is.On("SaveAuthorizationType", mock.Anything, mock.Anything).Return(nil)
	is.On("LoadAuthorizationType", mock.Anything).Return(domain.TypeOauth2, nil)

	op := &mocks.OAuthProvider{}
	op.On("OAuth2Configs", integration.GitHub).Return(&goauth2.Config{}, true)
	op.On("OAuth2Refresh", mock.Anything, mock.Anything).Return(nil)

	af := &domain.AuthorizationFactory{
		IntegrationsStorage:   is,
		AuthorizationsStorage: as,
		OAuthProvider:         op,
	}

	a1 := af.Create(1, integration.GitHub)
	at := goauth2.Token{
		AccessToken:  "123",
		RefreshToken: "456",
		Expiry:       time.Now().Add(-time.Hour),
		Extra:        nil,
	}
	b, err := json.Marshal(at)
	assert.NoError(t, err)
	a1.Data = b

	err = a1.Refresh()
	assert.NoError(t, err)

	err = as.Load(1, integration.GitHub, a1)
	assert.NoError(t, err)
	assert.NotEqual(t, []byte("{}"), a1.Data)
}

func TestRefresh_Oauth1(t *testing.T) {

	op := &mocks.OAuthProvider{}
	as := &mocks.AuthorizationsStorage{}

	is := &mocks.IntegrationsStorage{}
	is.On("SaveAuthorizationType", mock.Anything, mock.Anything).Return(nil)
	is.On("LoadAuthorizationType", mock.Anything).Return(domain.TypeOauth2, nil)

	af := &domain.AuthorizationFactory{
		IntegrationsStorage:   is,
		AuthorizationsStorage: as,
		OAuthProvider:         op,
	}

	a1 := af.Create(1, integration.GitHub)

	err := a1.Refresh()
	assert.NoError(t, err)
}

func TestRefresh_NotExpired(t *testing.T) {

	as := &mocks.AuthorizationsStorage{}
	op := &mocks.OAuthProvider{}

	is := &mocks.IntegrationsStorage{}
	is.On("SaveAuthorizationType", mock.Anything, mock.Anything).Return(nil)
	is.On("LoadAuthorizationType", mock.Anything).Return(domain.TypeOauth2, nil)

	af := &domain.AuthorizationFactory{
		IntegrationsStorage:   is,
		AuthorizationsStorage: as,
		OAuthProvider:         op,
	}

	a1 := af.Create(1, integration.GitHub)
	at := goauth2.Token{
		AccessToken:  "123",
		RefreshToken: "456",
		Expiry:       time.Now().Add(time.Hour * 24),
		Extra:        nil,
	}
	b, err := json.Marshal(at)
	assert.NoError(t, err)
	a1.Data = b

	err = a1.Refresh()
	assert.NoError(t, err)
}
