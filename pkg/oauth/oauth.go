package oauth

import (
	goauth2 "code.google.com/p/goauth2/oauth"
	"github.com/tambet/oauthplain"

	"github.com/toggl/pipes-api/pkg/integrations"
)

type Provider interface {
	OAuth2URL(integrations.ExternalServiceID) string
	OAuth1Configs(integrations.ExternalServiceID) (*oauthplain.Config, bool)
	OAuth1Exchange(integrations.ExternalServiceID, ParamsV1) ([]byte, error)
	OAuth2Exchange(integrations.ExternalServiceID, string) ([]byte, error)
	OAuth2Configs(integrations.ExternalServiceID) (*goauth2.Config, bool)
	OAuth2Refresh(*goauth2.Config, *goauth2.Token) error
}
