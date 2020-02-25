package environment

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"path/filepath"
	"sync"

	"code.google.com/p/goauth2/oauth"
	"github.com/tambet/oauthplain"
)

type Environment struct {
	envType string

	urls EnvUrls

	availableIntegrations   []*IntegrationConfig
	oAuth2Configs           map[string]*oauth.Config
	oAuth1Configs           map[string]*oauthplain.Config
	availableAuthorizations map[string]string

	mx sync.RWMutex
}

type EnvUrls struct {
	ReturnURL     map[string]string   `json:"return_url"`
	TogglAPIHost  map[string]string   `json:"toggl_api_host"`
	PipesAPIHost  map[string]string   `json:"pipes_api_host"`
	CorsWhitelist map[string][]string `json:"cors_whitelist"`
}

func New(envType, workDir string) *Environment {
	svc := &Environment{
		envType: envType,
		urls: EnvUrls{
			ReturnURL:     map[string]string{},
			TogglAPIHost:  map[string]string{},
			PipesAPIHost:  map[string]string{},
			CorsWhitelist: map[string][]string{},
		},
		availableIntegrations:   []*IntegrationConfig{},
		oAuth2Configs:           map[string]*oauth.Config{},
		oAuth1Configs:           map[string]*oauthplain.Config{},
		availableAuthorizations: map[string]string{},
	}

	svc.loadUrls(workDir)
	svc.loadIntegrations(workDir)
	svc.loadOauth2Configs(workDir)
	svc.loadOauth1Configs(workDir)
	svc.fillAuthorizations(svc.availableIntegrations)
	return svc
}

func (c *Environment) GetIntegrations() []*IntegrationConfig {
	c.mx.RLock()
	defer c.mx.RUnlock()
	return c.availableIntegrations
}

func (c *Environment) GetTogglAPIHost() string {
	c.mx.RLock()
	defer c.mx.RUnlock()
	return c.urls.TogglAPIHost[c.envType]
}

func (c *Environment) GetPipesAPIHost() string {
	c.mx.RLock()
	defer c.mx.RUnlock()
	return c.urls.PipesAPIHost[c.envType]
}

func (c *Environment) GetCorsWhitelist() []string {
	c.mx.RLock()
	defer c.mx.RUnlock()
	return c.urls.CorsWhitelist[c.envType]
}

func (c *Environment) OAuth2URL(service string) string {
	c.mx.RLock()
	defer c.mx.RUnlock()
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

	c.mx.RLock()
	config, res := c.oAuth1Configs[serviceID]
	if !res {
		c.mx.RUnlock()
		return nil, errors.New("service OAuth config not found")
	}
	c.mx.RUnlock()

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

	c.mx.RLock()
	config, res := c.oAuth2Configs[serviceID+"_"+c.envType]
	if !res {
		c.mx.RUnlock()
		return nil, errors.New("service OAuth config not found")
	}
	c.mx.RUnlock()

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
	c.mx.RLock()
	defer c.mx.RUnlock()
	v, found := c.oAuth1Configs[serviceID]
	return v, found
}

func (c *Environment) GetOAuth2Configs(serviceID string) (*oauth.Config, bool) {
	c.mx.RLock()
	defer c.mx.RUnlock()
	v, found := c.oAuth2Configs[serviceID+"_"+c.envType]
	return v, found
}

func (c *Environment) GetAvailableAuthorizations(serviceID string) string {
	c.mx.RLock()
	defer c.mx.RUnlock()
	return c.availableAuthorizations[serviceID]
}

func (c *Environment) loadIntegrations(workDir string) {
	c.mx.Lock()
	defer c.mx.Unlock()
	b, err := ioutil.ReadFile(filepath.Join(workDir, "config", "integrations.json"))
	if err != nil {
		log.Fatalf("Could not read integrations.json, reason: %v", err)
	}
	if err := json.Unmarshal(b, &c.availableIntegrations); err != nil {
		log.Fatalf("Could not parse integrations.json, reason: %v", err)
	}
}

func (c *Environment) loadOauth2Configs(workDir string) {
	c.mx.Lock()
	defer c.mx.Unlock()
	b, err := ioutil.ReadFile(filepath.Join(workDir, "config", "oauth2.json"))
	if err != nil {
		log.Fatalf("Could not read oauth2.json, reason: %v", err)
	}
	if err := json.Unmarshal(b, &c.oAuth2Configs); err != nil {
		log.Fatalf("Could not parse oauth2.json, reason: %v", err)
	}

}

func (c *Environment) loadOauth1Configs(workDir string) {
	c.mx.Lock()
	defer c.mx.Unlock()
	b, err := ioutil.ReadFile(filepath.Join(workDir, "config", "oauth1.json"))
	if err != nil {
		log.Fatalf("Could not read oauth1.json, reason: %v", err)
	}
	if err := json.Unmarshal(b, &c.oAuth1Configs); err != nil {
		log.Fatalf("Could not parse oauth1.json, reason: %v", err)
	}
}

func (c *Environment) loadUrls(workDir string) {
	c.mx.Lock()
	defer c.mx.Unlock()
	b, err := ioutil.ReadFile(filepath.Join(workDir, "config", "urls.json"))
	if err != nil {
		log.Fatalf("Could not read urls.json, reason: %v", err)
	}
	if err := json.Unmarshal(b, &c.urls); err != nil {
		log.Fatalf("Could not parse urls.json, reason: %v", err)
	}
}

func (c *Environment) fillAuthorizations(availableIntegrations []*IntegrationConfig) {
	c.mx.Lock()
	defer c.mx.Unlock()
	for _, integration := range availableIntegrations {
		c.availableAuthorizations[integration.ID] = integration.AuthType
	}
}
