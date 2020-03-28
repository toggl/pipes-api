package oauth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"sync"

	"code.google.com/p/goauth2/oauth"
	"github.com/tambet/oauthplain"

	"github.com/toggl/pipes-api/pkg/integration"
)

type InMemoryProvider struct {
	envType       string
	oauth1Config  io.Reader
	oauth2Config  io.Reader
	oAuth2Configs map[string]*oauth.Config
	oAuth1Configs map[string]*oauthplain.Config
	mx            sync.RWMutex
}

func NewInMemoryProvider(envType string, oauth1Config, oauth2Config io.Reader) *InMemoryProvider {
	svc := &InMemoryProvider{
		envType:       envType,
		oauth1Config:  oauth1Config,
		oauth2Config:  oauth2Config,
		oAuth2Configs: map[string]*oauth.Config{},
		oAuth1Configs: map[string]*oauthplain.Config{},
	}

	svc.loadOauth2Configs()
	svc.loadOauth1Configs()
	return svc
}

func (p *InMemoryProvider) OAuth2URL(sid integration.ID) string {
	p.mx.RLock()
	defer p.mx.RUnlock()
	config, ok := p.oAuth2Configs[string(sid)+"_"+p.envType]
	if !ok {
		return ""
	}
	return config.AuthCodeURL("__STATE__") + "&type=web_server"
}

func (p *InMemoryProvider) OAuth1Exchange(sid integration.ID, accountName, oAuthToken, oAuthVerifier string) (*oauthplain.Token, error) {
	if accountName == "" {
		return nil, errors.New("missing account_name")
	}
	if oAuthToken == "" {
		return nil, errors.New("missing oauth_token")
	}
	if oAuthVerifier == "" {
		return nil, errors.New("missing oauth_verifier")
	}

	p.mx.RLock()
	config, res := p.oAuth1Configs[string(sid)+"_"+p.envType]
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
	return token, nil
}

func (p *InMemoryProvider) OAuth2Exchange(sid integration.ID, code string) (*oauth.Token, error) {

	p.mx.RLock()
	config, res := p.oAuth2Configs[string(sid)+"_"+p.envType]
	if !res {
		p.mx.RUnlock()
		return nil, errors.New("OAuth config was not found for '" + string(sid) + "' service")
	}
	p.mx.RUnlock()

	transport := &oauth.Transport{Config: config}
	token, err := transport.Exchange(code)
	if err != nil {
		return nil, fmt.Errorf("could not get OAuth Access Token from '%s', reason: %e", config.TokenURL, err)
	}
	return token, nil
}

func (p *InMemoryProvider) OAuth1Configs(sid integration.ID) (*oauthplain.Config, bool) {
	p.mx.RLock()
	defer p.mx.RUnlock()
	v, found := p.oAuth1Configs[string(sid)+"_"+p.envType]
	return v, found
}

func (p *InMemoryProvider) OAuth2Configs(sid integration.ID) (*oauth.Config, bool) {
	p.mx.RLock()
	defer p.mx.RUnlock()
	v, found := p.oAuth2Configs[string(sid)+"_"+p.envType]
	return v, found
}

func (p *InMemoryProvider) OAuth2Refresh(cfg *oauth.Config, token *oauth.Token) error {
	transport := &oauth.Transport{Config: cfg, Token: token}
	return transport.Refresh()
}

func (p *InMemoryProvider) loadOauth2Configs() {
	p.mx.Lock()
	defer p.mx.Unlock()

	b, err := ioutil.ReadAll(p.oauth2Config)
	if err != nil {
		log.Fatalf("Could not read oauth2.json, reason: %v", err)
	}
	if err := json.Unmarshal(b, &p.oAuth2Configs); err != nil {
		log.Fatalf("Could not parse oauth2.json, reason: %v", err)
	}
}

func (p *InMemoryProvider) loadOauth1Configs() {
	p.mx.Lock()
	defer p.mx.Unlock()
	b, err := ioutil.ReadAll(p.oauth1Config)
	if err != nil {
		log.Fatalf("Could not read oauth1.json, reason: %v", err)
	}
	if err := json.Unmarshal(b, &p.oAuth1Configs); err != nil {
		log.Fatalf("Could not parse oauth1.json, reason: %v", err)
	}
}
