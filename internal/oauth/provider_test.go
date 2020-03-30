package oauth

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"code.google.com/p/goauth2/oauth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/toggl/pipes-api/pkg/domain"
)

func TestNewProvider(t *testing.T) {
	p, err := Create("development", getOauth1ConfigForTests(""), getOauth2ConfigForTests(""))
	require.NoError(t, err)

	assert.Equal(t, "development", p.envType)
	assert.NotNil(t, p.oAuth1Configs)
	assert.NotNil(t, p.oAuth2Configs)
	assert.Equal(t, 1, len(p.oAuth1Configs))
	assert.Equal(t, 1, len(p.oAuth2Configs))

	t.Run("Wrong OAuth2 Reader", func(t *testing.T) {
		p, err := Create("development", &errReader{}, getOauth2ConfigForTests(""))
		assert.Nil(t, p)
		assert.Error(t, err)
	})

	t.Run("Wrong OAuth1 Reader", func(t *testing.T) {
		p, err := Create("development", getOauth1ConfigForTests(""), &errReader{})
		assert.Nil(t, p)
		assert.Error(t, err)
	})

	t.Run("Wrong OAuth2 Format", func(t *testing.T) {
		p, err := Create("development", getOauth1ConfigForTests(""), strings.NewReader(""))
		assert.Nil(t, p)
		assert.Error(t, err)
	})

	t.Run("Wrong OAuth1 Format", func(t *testing.T) {
		p, err := Create("development", strings.NewReader(""), getOauth2ConfigForTests(""))
		assert.Nil(t, p)
		assert.Error(t, err)
	})
}

func TestProvider_OAuth1Configs(t *testing.T) {
	p, err := Create("development", getOauth1ConfigForTests(""), getOauth2ConfigForTests(""))
	require.NoError(t, err)

	c, exists := p.OAuth1Configs(domain.FreshBooks)
	assert.True(t, exists)
	assert.NotNil(t, c)
}

func TestProvider_OAuth1Exchange(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "appication/json")
			w.Write([]byte(`oauth_token=valid&oauth_token_secret=secret`))
		}))

		p, err := Create("development", getOauth1ConfigForTests(ts.URL), getOauth2ConfigForTests(ts.URL))
		require.NoError(t, err)

		token, err := p.OAuth1Exchange(domain.FreshBooks, "client", "token", "verifier")
		assert.Nil(t, err)
		assert.Equal(t, "valid", token.OAuthToken)
		assert.Equal(t, "secret", token.OAuthTokenSecret)
		assert.Equal(t, "client", token.Extra["account_name"])
	})

	t.Run("Error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(401)
		}))

		p, err := Create("development", getOauth1ConfigForTests(ts.URL), getOauth2ConfigForTests(ts.URL))
		require.NoError(t, err)

		token, err := p.OAuth1Exchange(domain.FreshBooks, "client", "token", "verifier")
		assert.Error(t, err)
		assert.Nil(t, token)
	})

	t.Run("Missing parameter", func(t *testing.T) {
		p, err := Create("development", getOauth1ConfigForTests(""), getOauth2ConfigForTests(""))
		require.NoError(t, err)

		_, err = p.OAuth1Exchange(domain.FreshBooks, "", "secret", "verifier")
		assert.Error(t, err)
		_, err = p.OAuth1Exchange(domain.FreshBooks, "token", "", "verifier")
		assert.Error(t, err)
		_, err = p.OAuth1Exchange(domain.FreshBooks, "token", "secret", "")
		assert.Error(t, err)
	})

	t.Run("Unknown Service", func(t *testing.T) {
		p, err := Create("development", getOauth1ConfigForTests(""), getOauth2ConfigForTests(""))
		require.NoError(t, err)

		token, err := p.OAuth1Exchange("unknown", "toggl", "token", "verifier")
		assert.Error(t, err)
		assert.Nil(t, token)
	})
}

func TestProvider_OAuth2Configs(t *testing.T) {
	p, err := Create("development", getOauth1ConfigForTests(""), getOauth2ConfigForTests(""))
	require.NoError(t, err)

	c, exists := p.OAuth2Configs(domain.GitHub)
	assert.True(t, exists)
	assert.NotNil(t, c)
}

func TestProvider_OAuth2Exchange(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "appication/json")
			w.Write([]byte(`{"access_token":"valid", "refresh_token":"test"}`))
		}))

		p, err := Create("development", getOauth1ConfigForTests(ts.URL), getOauth2ConfigForTests(ts.URL))
		require.NoError(t, err)

		token, err := p.OAuth2Exchange(domain.GitHub, "test_code")
		assert.Nil(t, err)
		assert.Equal(t, "valid", token.AccessToken)
		assert.Equal(t, "test", token.RefreshToken)
	})

	t.Run("Error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(401)
		}))

		p, err := Create("development", getOauth1ConfigForTests(ts.URL), getOauth2ConfigForTests(ts.URL))
		require.NoError(t, err)

		token, err := p.OAuth2Exchange(domain.GitHub, "test_code")
		assert.Error(t, err)
		assert.Nil(t, token)
	})

	t.Run("Unknown Service", func(t *testing.T) {
		p, err := Create("development", getOauth1ConfigForTests(""), getOauth2ConfigForTests(""))
		require.NoError(t, err)

		token, err := p.OAuth2Exchange("unknown", "test_code")
		assert.Error(t, err)
		assert.Nil(t, token)
	})
}

func TestProvider_OAuth2Refresh(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "appication/json")
		w.Write([]byte(`{"access_token":"valid", "refresh_token":"test"}`))
	}))

	p, err := Create("development", getOauth1ConfigForTests(ts.URL), getOauth2ConfigForTests(ts.URL))
	require.NoError(t, err)

	token := &oauth.Token{
		AccessToken:  "expired",
		RefreshToken: "update_token",
		Expiry:       time.Time{},
		Extra:        nil,
	}

	cfg, ok := p.OAuth2Configs(domain.GitHub)
	assert.True(t, ok)

	err = p.OAuth2Refresh(cfg, token)
	assert.Nil(t, err)
	assert.Equal(t, "valid", token.AccessToken)
	assert.Equal(t, "test", token.RefreshToken)
}

func TestProvider_OAuth2URL(t *testing.T) {
	p, err := Create("development", getOauth1ConfigForTests(""), getOauth2ConfigForTests(""))
	require.NoError(t, err)

	url := p.OAuth2URL(domain.GitHub)
	assert.Equal(t, "/authorize?access_type=&approval_prompt=&client_id=123&redirect_uri=&response_type=code&state=__STATE__&type=web_server", url)

	url2 := p.OAuth2URL("unknown")
	assert.Equal(t, "", url2)
}

func getOauth1ConfigForTests(url string) io.Reader {
	str := `
{
	"freshbooks_development": {
	  "ConsumerKey": "123",
	  "ConsumerSecret": "456",
	  "RequestTokenUrl": "` + url + `/%s/request_token",
	  "AuthorizeTokenUrl": "` + url + `/%s/authorize_token",
	  "AccessTokenUrl": "` + url + `/%s/access_token"
	}
}
`
	return strings.NewReader(str)
}

func getOauth2ConfigForTests(url string) io.Reader {
	str := `
{
	"github_development": {
		"ClientId": "123",
		"ClientSecret": "456",
		"AuthURL": "` + url + `/authorize",
		"TokenURL": "` + url + `/token",
		"RedirectURL": "` + url + `"
	}
}
`
	return strings.NewReader(str)
}

type errReader struct{}

func (r errReader) Read(p []byte) (n int, err error) {
	return 0, errors.New("error")
}
