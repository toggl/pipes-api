package service

import (
	goauth2 "code.google.com/p/goauth2/oauth"
	"github.com/tambet/oauthplain"

	"github.com/toggl/pipes-api/pkg/integration"
)

//go:generate mockery -name OAuthProvider -case underscore -inpkg
type OAuthProvider interface {
	OAuth2URL(integration.ID) string
	OAuth1Configs(integration.ID) (*oauthplain.Config, bool)
	OAuth1Exchange(sid integration.ID, accountName, oAuthToken, oAuthVerifier string) (*oauthplain.Token, error)
	OAuth2Exchange(sid integration.ID, code string) (*goauth2.Token, error)
	OAuth2Configs(integration.ID) (*goauth2.Config, bool)
	OAuth2Refresh(*goauth2.Config, *goauth2.Token) error
}
