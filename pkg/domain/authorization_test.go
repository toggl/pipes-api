package domain_test

import (
	"testing"
	"time"

	goauth2 "code.google.com/p/goauth2/oauth"
	"github.com/stretchr/testify/assert"
	"github.com/tambet/oauthplain"

	"github.com/toggl/pipes-api/pkg/domain"
	"github.com/toggl/pipes-api/pkg/integration"
)

func TestNewAuthorization(t *testing.T) {
	a := domain.NewAuthorization(1, integration.GitHub)
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

	a := domain.NewAuthorization(1, "test")
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

	a := domain.NewAuthorization(1, "test")
	err := a.SetOAuth1Token(&token)
	assert.NoError(t, err)

	assert.Equal(t, `{"ConsumerKey":"test","ConsumerSecret":"test","OAuthToken":"test","OAuthTokenSecret":"test","OAuthVerifier":"test","AuthorizeUrl":"test","Extra":null}`, string(a.Data))
}
