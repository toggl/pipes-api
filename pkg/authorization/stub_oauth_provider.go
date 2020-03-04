package authorization

import (
	"code.google.com/p/goauth2/oauth"

	"github.com/toggl/pipes-api/pkg/integrations"
)

type stubOauthProvider struct {
	NotFound     bool
	RefreshError bool
}

func (sp *stubOauthProvider) GetOAuth2Configs(id integrations.ExternalServiceID) (*oauth.Config, bool) {
	if sp.NotFound {
		return nil, false
	}

	return &oauth.Config{
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

func (sp *stubOauthProvider) Refresh(*oauth.Config, *oauth.Token) error {
	if sp.RefreshError {
		return oauth.OAuthError{}
	}
	return nil
}
