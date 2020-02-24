package environment

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"path/filepath"

	"code.google.com/p/goauth2/oauth"
	"github.com/tambet/oauthplain"
)

type Environment struct {
	envType string

	urls struct {
		ReturnURL     map[string]string   `json:"return_url"`
		TogglAPIHost  map[string]string   `json:"toggl_api_host"`
		PipesAPIHost  map[string]string   `json:"pipes_api_host"`
		CorsWhitelist map[string][]string `json:"cors_whitelist"`
	}

	availableIntegrations   []*IntegrationConfig
	oAuth2Configs           map[string]*oauth.Config
	oAuth1Configs           map[string]*oauthplain.Config
	availableAuthorizations map[string]string
}

func New(envType, workDir string) *Environment {
	svc := &Environment{envType: envType}
	svc.loadUrls(workDir)
	svc.loadIntegrations(workDir)
	svc.loadOauth2Configs(workDir)
	svc.loadOauth1Configs(workDir)
	svc.fillAuthorizations(svc.availableIntegrations)
	return svc
}

func (c *Environment) loadIntegrations(workDir string) {
	b, err := ioutil.ReadFile(filepath.Join(workDir, "config", "integrations.json"))
	if err != nil {
		log.Fatal(err)
	}
	if err := json.Unmarshal(b, &c.availableIntegrations); err != nil {
		log.Fatal(err)
	}
}

func (c *Environment) loadOauth2Configs(workDir string) {
	b, err := ioutil.ReadFile(filepath.Join(workDir, "config", "oauth2.json"))
	if err != nil {
		log.Fatal(err)
	}
	if err := json.Unmarshal(b, &c.oAuth2Configs); err != nil {
		log.Fatal(err)
	}

}

func (c *Environment) loadOauth1Configs(workDir string) {
	b, err := ioutil.ReadFile(filepath.Join(workDir, "config", "oauth1.json"))
	if err != nil {
		log.Fatal(err)
	}
	if err := json.Unmarshal(b, &c.oAuth1Configs); err != nil {
		log.Fatal(err)
	}
}

func (c *Environment) fillAuthorizations(availableIntegrations []*IntegrationConfig) {
	for _, integration := range availableIntegrations {
		c.availableAuthorizations[integration.ID] = integration.AuthType
	}
}

func (c *Environment) GetIntegrations() []*IntegrationConfig {
	return c.availableIntegrations
}

func (c *Environment) loadUrls(workDir string) {
	b, err := ioutil.ReadFile(filepath.Join(workDir, "config", "urls.json"))
	if err != nil {
		log.Fatal(err)
	}
	if err := json.Unmarshal(b, &c.urls); err != nil {
		log.Fatal(err)
	}
}

func (c *Environment) GetTogglAPIHost() string {
	return c.urls.TogglAPIHost[c.envType]
}

func (c *Environment) GetPipesAPIHost() string {
	return c.urls.PipesAPIHost[c.envType]
}

func (c *Environment) GetCorsWhitelist() []string {
	return c.urls.CorsWhitelist[c.envType]
}

func (c *Environment) OAuth2URL(service string) string {
	config, ok := c.oAuth2Configs[service+"_"+c.envType]
	if !ok {
		return ""
	}
	return config.AuthCodeURL("__STATE__") + "&type=web_server"
}

func (c *Environment) OAuth1Exchange(serviceID string, payload map[string]interface{}) ([]byte, error) {
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
	config, res := c.oAuth1Configs[serviceID]
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

func (c *Environment) OAuth2Exchange(serviceID string, payload map[string]interface{}) ([]byte, error) {
	code := payload["code"].(string)
	if code == "" {
		return nil, errors.New("missing code")
	}
	config, res := c.oAuth2Configs[serviceID+"_"+c.envType]
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

func (c *Environment) GetOAuth1Configs(serviceID string) (*oauthplain.Config, bool) {
	v, found := c.oAuth1Configs[serviceID]
	return v, found
}

func (c *Environment) GetOAuth2Configs(serviceID string) (*oauth.Config, bool) {
	v, found := c.oAuth2Configs[serviceID+"_"+c.envType]
	return v, found
}

func (c *Environment) GetAvailableAuthorizations(serviceID string) string {
	return c.availableAuthorizations[serviceID]
}
