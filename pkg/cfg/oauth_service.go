package cfg

import (
	"encoding/json"
	"errors"

	"code.google.com/p/goauth2/oauth"
	"github.com/tambet/oauthplain"
)

type OAuthService struct {
	Environment string
}

func (os *OAuthService) OAuth2URL(service string) string {
	config, ok := OAuth2Configs[service+"_"+os.Environment]
	if !ok {
		return ""
	}
	return config.AuthCodeURL("__STATE__") + "&type=web_server"
}

func (os *OAuthService) OAuth1Exchange(serviceID string, payload map[string]interface{}) ([]byte, error) {
	accountName := payload["account_name"].(string)
	if accountName == "" {
		return nil, errors.New("missing account_name")
	}
	oAuthToken := payload["oauth_token"].(string)
	if oAuthToken == "" {
		return nil, errors.New("missing oauth_token")
	}
	oAuthVerifier := payload["oauth_verifier"].(string)
	if oAuthVerifier == "" {
		return nil, errors.New("missing oauth_verifier")
	}
	config, res := OAuth1Configs[serviceID]
	if !res {
		return nil, errors.New("service OAuth config not found")
	}
	transport := &oauthplain.Transport{
		Config: config.UpdateURLs(accountName),
	}
	token := &oauthplain.Token{
		OAuthToken:    oAuthToken,
		OAuthVerifier: oAuthVerifier,
	}
	if err := transport.Exchange(token); err != nil {
		return nil, err
	}
	token.Extra = make(map[string]string)
	token.Extra["account_name"] = accountName
	b, err := json.Marshal(token)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (os *OAuthService) OAuth2Exchange(serviceID string, payload map[string]interface{}) ([]byte, error) {
	code := payload["code"].(string)
	if code == "" {
		return nil, errors.New("missing code")
	}
	config, res := OAuth2Configs[serviceID+"_"+os.Environment]
	if !res {
		return nil, errors.New("service OAuth config not found")
	}
	transport := &oauth.Transport{Config: config}
	token, err := transport.Exchange(code)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(token)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (os *OAuthService) GetOAuth1Configs(serviceID string) (*oauthplain.Config, bool) {
	v, found := OAuth1Configs[serviceID]
	return v, found
}

func (os *OAuthService) GetAvailableAuthorizations(serviceID string) string {
	return AvailableAuthorizations[serviceID]
}
