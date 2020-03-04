package oauth

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"sync"

	"code.google.com/p/goauth2/oauth"
	"github.com/tambet/oauthplain"

	"github.com/toggl/pipes-api/pkg/integrations"
)

type Provider struct {
	EnvType          string
	Oauth1ConfigPath string
	Oauth2ConfigPath string

	oAuth2Configs map[string]*oauth.Config
	oAuth1Configs map[string]*oauthplain.Config
	mx            sync.RWMutex
}

func NewProvider(envType, oauth1ConfigPath, oauth2ConfigPath string) *Provider {
	svc := &Provider{
		EnvType:          envType,
		Oauth1ConfigPath: oauth1ConfigPath,
		Oauth2ConfigPath: oauth2ConfigPath,
		oAuth2Configs:    map[string]*oauth.Config{},
		oAuth1Configs:    map[string]*oauthplain.Config{},
	}

	svc.loadOauth2Configs()
	svc.loadOauth1Configs()
	return svc
}

func (p *Provider) GetOAuth2URL(externalServiceID integrations.ExternalServiceID) string {
	p.mx.RLock()
	defer p.mx.RUnlock()
	config, ok := p.oAuth2Configs[string(externalServiceID)+"_"+p.EnvType]
	if !ok {
		return ""
	}
	return config.AuthCodeURL("__STATE__") + "&type=web_server"
}

func (p *Provider) OAuth1Exchange(externalServiceID integrations.ExternalServiceID, payload map[string]interface{}) ([]byte, error) {
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

	p.mx.RLock()
	config, res := p.oAuth1Configs[string(externalServiceID)]
	if !res {
		p.mx.RUnlock()
		return nil, errors.New("service OAuth config not found")
	}
	p.mx.RUnlock()

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

func (p *Provider) OAuth2Exchange(externalServiceID integrations.ExternalServiceID, payload map[string]interface{}) ([]byte, error) {
	code := payload["code"].(string)
	if code == "" {
		return nil, errors.New("missing code")
	}

	p.mx.RLock()
	config, res := p.oAuth2Configs[string(externalServiceID)+"_"+p.EnvType]
	if !res {
		p.mx.RUnlock()
		return nil, errors.New("service OAuth config not found")
	}
	p.mx.RUnlock()

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

func (p *Provider) GetOAuth1Configs(externalServiceID integrations.ExternalServiceID) (*oauthplain.Config, bool) {
	p.mx.RLock()
	defer p.mx.RUnlock()
	v, found := p.oAuth1Configs[string(externalServiceID)]
	return v, found
}

func (p *Provider) GetOAuth2Configs(externalServiceID integrations.ExternalServiceID) (*oauth.Config, bool) {
	p.mx.RLock()
	defer p.mx.RUnlock()
	v, found := p.oAuth2Configs[string(externalServiceID)+"_"+p.EnvType]
	return v, found
}

func (p *Provider) Refresh(config *oauth.Config, token *oauth.Token) error {
	transport := &oauth.Transport{Config: config, Token: token}
	return transport.Refresh()
}

func (p *Provider) loadOauth2Configs() {
	p.mx.Lock()
	defer p.mx.Unlock()

	b, err := ioutil.ReadFile(p.Oauth2ConfigPath)
	if err != nil {
		log.Fatalf("Could not read oauth2.json, reason: %v", err)
	}
	if err := json.Unmarshal(b, &p.oAuth2Configs); err != nil {
		log.Fatalf("Could not parse oauth2.json, reason: %v", err)
	}
}

func (p *Provider) loadOauth1Configs() {
	p.mx.Lock()
	defer p.mx.Unlock()
	b, err := ioutil.ReadFile(p.Oauth1ConfigPath)
	if err != nil {
		log.Fatalf("Could not read oauth1.json, reason: %v", err)
	}
	if err := json.Unmarshal(b, &p.oAuth1Configs); err != nil {
		log.Fatalf("Could not parse oauth1.json, reason: %v", err)
	}
}
