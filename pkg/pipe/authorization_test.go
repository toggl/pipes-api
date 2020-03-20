package pipe

import (
	"testing"
	"time"

	"code.google.com/p/goauth2/oauth"
	"github.com/stretchr/testify/assert"

	"github.com/toggl/pipes-api/pkg/integrations"
)

func TestNewAuthorization(t *testing.T) {
	a := NewAuthorization(1, integrations.GitHub, "")
	assert.Equal(t, 1, a.WorkspaceID)
	assert.Equal(t, integrations.GitHub, a.ServiceID)
	assert.NotNil(t, a.Data)
}

func TestSetOauth2Token(t *testing.T) {
	token := oauth.Token{
		AccessToken:  "test",
		RefreshToken: "test",
		Expiry:       time.Time{},
		Extra:        nil,
	}

	a := NewAuthorization(1, "test", "")
	err := a.SetOAuth2Token(&token)
	assert.NoError(t, err)

	assert.Equal(t, `{"AccessToken":"test","RefreshToken":"test","Expiry":"0001-01-01T00:00:00Z","Extra":null}`, string(a.Data))
}
