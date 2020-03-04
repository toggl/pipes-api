package authorization

import (
	"testing"
	"time"

	"code.google.com/p/goauth2/oauth"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	a := New(1, "test")
	assert.Equal(t, 1, a.WorkspaceID)
	assert.Equal(t, "test", a.ServiceID)
	assert.NotNil(t, a.Data)
}

func TestSetOauth2Token(t *testing.T) {
	token := oauth.Token{
		AccessToken:  "test",
		RefreshToken: "test",
		Expiry:       time.Time{},
		Extra:        nil,
	}

	a := New(1, "test")
	err := a.SetOauth2Token(token)
	assert.NoError(t, err)

	assert.Equal(t, `{"AccessToken":"test","RefreshToken":"test","Expiry":"0001-01-01T00:00:00Z","Extra":null}`, string(a.Data))
}
