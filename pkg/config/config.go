package config

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"path/filepath"
	"sync"

	"code.google.com/p/goauth2/oauth"
	"github.com/tambet/oauthplain"

	"github.com/toggl/pipes-api/pkg/integrations"
)

type Config struct {
	WorkDir string
	EnvType string
	Urls    EnvUrls

	oAuth2Configs map[string]*oauth.Config
	oAuth1Configs map[string]*oauthplain.Config
	mx            sync.RWMutex
}

type EnvUrls struct {
	ReturnURL     map[string]string   `json:"return_url"`
	TogglAPIHost  map[string]string   `json:"toggl_api_host"`
	PipesAPIHost  map[string]string   `json:"pipes_api_host"`
	CorsWhitelist map[string][]string `json:"cors_whitelist"`
}

func Load(envType, workDir string) *Config {
	svc := &Config{
		EnvType: envType,
		WorkDir: workDir,
		Urls: EnvUrls{
			ReturnURL:     map[string]string{},
			TogglAPIHost:  map[string]string{},
			PipesAPIHost:  map[string]string{},
			CorsWhitelist: map[string][]string{},
		},

		oAuth2Configs: map[string]*oauth.Config{},
		oAuth1Configs: map[string]*oauthplain.Config{},
	}

	svc.loadUrls()
	svc.loadOauth2Configs()
	svc.loadOauth1Configs()
	return svc
}

func (c *Config) GetTogglAPIHost() string {
	c.mx.RLock()
	defer c.mx.RUnlock()
	return c.Urls.TogglAPIHost[c.EnvType]
}

func (c *Config) GetOAuth2URL(externalServiceID integrations.ExternalServiceID) string {
	c.mx.RLock()
	defer c.mx.RUnlock()
	config, ok := c.oAuth2Configs[string(externalServiceID)+"_"+c.EnvType]
	if !ok {
		return ""
	}
	return config.AuthCodeURL("__STATE__") + "&type=web_server"
}

func (c *Config) OAuth1Exchange(externalServiceID integrations.ExternalServiceID, payload map[string]interface{}) ([]byte, error) {
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
	config, res := c.oAuth1Configs[string(externalServiceID)]
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

func (c *Config) OAuth2Exchange(externalServiceID integrations.ExternalServiceID, payload map[string]interface{}) ([]byte, error) {
	code := payload["code"].(string)
	if code == "" {
		return nil, errors.New("missing code")
	}

	c.mx.RLock()
	config, res := c.oAuth2Configs[string(externalServiceID)+"_"+c.EnvType]
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

func (c *Config) GetOAuth1Configs(externalServiceID integrations.ExternalServiceID) (*oauthplain.Config, bool) {
	c.mx.RLock()
	defer c.mx.RUnlock()
	v, found := c.oAuth1Configs[string(externalServiceID)]
	return v, found
}

func (c *Config) GetOAuth2Configs(externalServiceID integrations.ExternalServiceID) (*oauth.Config, bool) {
	c.mx.RLock()
	defer c.mx.RUnlock()
	v, found := c.oAuth2Configs[string(externalServiceID)+"_"+c.EnvType]
	return v, found
}

func (c *Config) Refresh(config *oauth.Config, token *oauth.Token) error {
	transport := &oauth.Transport{Config: config, Token: token}
	return transport.Refresh()
}

func (c *Config) loadOauth2Configs() {
	c.mx.Lock()
	defer c.mx.Unlock()
	b, err := ioutil.ReadFile(filepath.Join(c.WorkDir, "config", "oauth2.json"))
	if err != nil {
		log.Fatalf("Could not read oauth2.json, reason: %v", err)
	}
	if err := json.Unmarshal(b, &c.oAuth2Configs); err != nil {
		log.Fatalf("Could not parse oauth2.json, reason: %v", err)
	}

}

func (c *Config) loadOauth1Configs() {
	c.mx.Lock()
	defer c.mx.Unlock()
	b, err := ioutil.ReadFile(filepath.Join(c.WorkDir, "config", "oauth1.json"))
	if err != nil {
		log.Fatalf("Could not read oauth1.json, reason: %v", err)
	}
	if err := json.Unmarshal(b, &c.oAuth1Configs); err != nil {
		log.Fatalf("Could not parse oauth1.json, reason: %v", err)
	}
}

func (c *Config) loadUrls() {
	c.mx.Lock()
	defer c.mx.Unlock()
	b, err := ioutil.ReadFile(filepath.Join(c.WorkDir, "config", "urls.json"))
	if err != nil {
		log.Fatalf("Could not read urls.json, reason: %v", err)
	}
	if err := json.Unmarshal(b, &c.Urls); err != nil {
		log.Fatalf("Could not parse urls.json, reason: %v", err)
	}
}
