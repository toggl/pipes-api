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

type ParamsV1 struct {
	AccountName string `json:"account_name,omitempty"`
	Token       string `json:"oauth_token,omitempty"`
	Verifier    string `json:"oauth_verifier,omitempty"`
}

type Provider struct {
	envType          string
	oauth1ConfigPath string
	oauth2ConfigPath string
	oAuth2Configs    map[string]*oauth.Config
	oAuth1Configs    map[string]*oauthplain.Config
	mx               sync.RWMutex
}

func NewProvider(envType, oauth1ConfigPath, oauth2ConfigPath string) *Provider {
	svc := &Provider{
		envType:          envType,
		oauth1ConfigPath: oauth1ConfigPath,
		oauth2ConfigPath: oauth2ConfigPath,
		oAuth2Configs:    map[string]*oauth.Config{},
		oAuth1Configs:    map[string]*oauthplain.Config{},
	}

	svc.loadOauth2Configs()
	svc.loadOauth1Configs()
	return svc
}

func (p *Provider) OAuth2URL(sid integrations.ExternalServiceID) string {
	p.mx.RLock()
	defer p.mx.RUnlock()
	config, ok := p.oAuth2Configs[string(sid)+"_"+p.envType]
	if !ok {
		return ""
	}
	return config.AuthCodeURL("__STATE__") + "&type=web_server"
}

func (p *Provider) OAuth1Exchange(sid integrations.ExternalServiceID, op ParamsV1) ([]byte, error) {
	if op.AccountName == "" {
		return nil, errors.New("missing account_name")
	}
	if op.Token == "" {
		return nil, errors.New("missing oauth_token")
	}
	if op.Verifier == "" {
		return nil, errors.New("missing oauth_verifier")
	}

	p.mx.RLock()
	config, res := p.oAuth1Configs[string(sid)]
	if !res {
		p.mx.RUnlock()
		return nil, errors.New("service OAuth config not found")
	}
	p.mx.RUnlock()

	transport := &oauthplain.Transport{
		Config: config.UpdateURLs(op.AccountName),
	}
	token := &oauthplain.Token{
		OAuthToken:    op.Token,
		OAuthVerifier: op.Verifier,
	}
	if err := transport.Exchange(token); err != nil {
		return nil, err
	}
	token.Extra = make(map[string]string)
	token.Extra["account_name"] = op.AccountName
	b, err := json.Marshal(token)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (p *Provider) OAuth2Exchange(sid integrations.ExternalServiceID, code string) ([]byte, error) {

	p.mx.RLock()
	config, res := p.oAuth2Configs[string(sid)+"_"+p.envType]
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

func (p *Provider) OAuth1Configs(sid integrations.ExternalServiceID) (*oauthplain.Config, bool) {
	p.mx.RLock()
	defer p.mx.RUnlock()
	v, found := p.oAuth1Configs[string(sid)]
	return v, found
}

func (p *Provider) OAuth2Configs(sid integrations.ExternalServiceID) (*oauth.Config, bool) {
	p.mx.RLock()
	defer p.mx.RUnlock()
	v, found := p.oAuth2Configs[string(sid)+"_"+p.envType]
	return v, found
}

func (p *Provider) OAuth2Refresh(cfg *oauth.Config, token *oauth.Token) error {
	transport := &oauth.Transport{Config: cfg, Token: token}
	return transport.Refresh()
}

func (p *Provider) loadOauth2Configs() {
	p.mx.Lock()
	defer p.mx.Unlock()

	b, err := ioutil.ReadFile(p.oauth2ConfigPath)
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
	b, err := ioutil.ReadFile(p.oauth1ConfigPath)
	if err != nil {
		log.Fatalf("Could not read oauth1.json, reason: %v", err)
	}
	if err := json.Unmarshal(b, &p.oAuth1Configs); err != nil {
		log.Fatalf("Could not parse oauth1.json, reason: %v", err)
	}
}
