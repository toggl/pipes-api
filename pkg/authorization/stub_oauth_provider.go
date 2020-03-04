package authorization

import "code.google.com/p/goauth2/oauth"

type stubOauthProvider struct {
	NotFound     bool
	RefreshError bool
}

func (sp *stubOauthProvider) GetOAuth2Configs(externalServiceID string) (*oauth.Config, bool) {
	if sp.NotFound {
		return nil, false
	}

	return &oauth.Config{
		ClientId:       externalServiceID,
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
