package oauth

import (
	goauth2 "code.google.com/p/goauth2/oauth"
	"github.com/tambet/oauthplain"

	"github.com/toggl/pipes-api/pkg/integrations"
)

type StubProvider struct {
	NotFound     bool
	RefreshError bool
}

func (sp *StubProvider) OAuth2URL(integrations.ExternalServiceID) string {
	return ""
}

func (sp *StubProvider) OAuth1Configs(integrations.ExternalServiceID) (*oauthplain.Config, bool) {
	return &oauthplain.Config{
		ConsumerKey:       "",
		ConsumerSecret:    "",
		RequestTokenUrl:   "",
		AuthorizeTokenUrl: "",
		AccessTokenUrl:    "",
	}, true
}

func (sp *StubProvider) OAuth1Exchange(integrations.ExternalServiceID, ParamsV1) ([]byte, error) {
	return []byte{}, nil
}

func (sp *StubProvider) OAuth2Exchange(integrations.ExternalServiceID, string) ([]byte, error) {
	return []byte{}, nil
}

func (sp *StubProvider) OAuth2Configs(id integrations.ExternalServiceID) (*goauth2.Config, bool) {
	return &goauth2.Config{
		ClientId:       string(id),
		ClientSecret:   "",
		Scope:          "",
		AuthURL:        "http://localhost/",
		TokenURL:       "http://localhost/",
		RedirectURL:    "http://localhost/",
		TokenCache:     nil,
		AccessType:     "",
		ApprovalPrompt: "",
	}, true
}

func (sp *StubProvider) OAuth2Refresh(*goauth2.Config, *goauth2.Token) error {
	if sp.RefreshError {
		return goauth2.OAuthError{}
	}
	return nil
}
