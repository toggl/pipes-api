package oauth

import (
	goauth2 "code.google.com/p/goauth2/oauth"
	"github.com/tambet/oauthplain"

	"github.com/toggl/pipes-api/pkg/integrations"
)

//go:generate mockery -name Provider -case underscore -inpkg
type Provider interface {
	OAuth2URL(integrations.ExternalServiceID) string
	OAuth1Configs(integrations.ExternalServiceID) (*oauthplain.Config, bool)
	OAuth1Exchange(sid integrations.ExternalServiceID, accountName, oAuthToken, oAuthVerifier string) (*oauthplain.Token, error)
	OAuth2Exchange(sid integrations.ExternalServiceID, code string) (*goauth2.Token, error)
	OAuth2Configs(integrations.ExternalServiceID) (*goauth2.Config, bool)
	OAuth2Refresh(*goauth2.Config, *goauth2.Token) error
}
