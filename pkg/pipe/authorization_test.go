package pipe

import (
	"testing"
	"time"

	"code.google.com/p/goauth2/oauth"
	"github.com/stretchr/testify/assert"

	"github.com/toggl/pipes-api/pkg/integration"
)

func TestNewAuthorization(t *testing.T) {
	af := AuthorizationFactory{
		IntegrationsStorage:   &MockIntegrationsStorage{},
		AuthorizationsStorage: &MockAuthorizationsStorage{},
		OAuthProvider:         &MockOAuthProvider{},
	}
	a := af.Create(1, integration.GitHub)
	assert.Equal(t, 1, a.WorkspaceID)
	assert.Equal(t, integration.GitHub, a.ServiceID)
	assert.NotNil(t, a.Data)
}

func TestSetOauth2Token(t *testing.T) {
	token := oauth.Token{
		AccessToken:  "test",
		RefreshToken: "test",
		Expiry:       time.Time{},
		Extra:        nil,
	}

	af := AuthorizationFactory{
		IntegrationsStorage:   &MockIntegrationsStorage{},
		AuthorizationsStorage: &MockAuthorizationsStorage{},
		OAuthProvider:         &MockOAuthProvider{},
	}

	a := af.Create(1, "test")
	err := a.SetOAuth2Token(&token)
	assert.NoError(t, err)

	assert.Equal(t, `{"AccessToken":"test","RefreshToken":"test","Expiry":"0001-01-01T00:00:00Z","Extra":null}`, string(a.Data))
}
