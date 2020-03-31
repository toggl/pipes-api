package oauthprovider

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"sync"

	"code.google.com/p/goauth2/oauth"
	"github.com/tambet/oauthplain"

	"github.com/toggl/pipes-api/pkg/domain"
)

type Provider struct {
	envType       string
	oAuth2Configs map[string]*oauth.Config
	oAuth1Configs map[string]*oauthplain.Config
	mx            sync.RWMutex
}

func NewProvider(envType string, oauth1Config, oauth2Config io.Reader) (*Provider, error) {
	svc := &Provider{
		envType:       envType,
		oAuth2Configs: map[string]*oauth.Config{},
		oAuth1Configs: map[string]*oauthplain.Config{},
	}

	if err := svc.loadOauth2Configs(oauth2Config); err != nil {
		return nil, err
	}
	if err := svc.loadOauth1Configs(oauth1Config); err != nil {
		return nil, err
	}
	return svc, nil
}

func (p *Provider) OAuth2URL(sid domain.IntegrationID) string {
	p.mx.RLock()
	defer p.mx.RUnlock()
	config, ok := p.oAuth2Configs[string(sid)+"_"+p.envType]
	if !ok {
		return ""
	}
	return config.AuthCodeURL("__STATE__") + "&type=web_server"
}

func (p *Provider) OAuth1Exchange(sid domain.IntegrationID, accountName, oAuthToken, oAuthVerifier string) (*oauthplain.Token, error) {
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

func (p *Provider) OAuth2Exchange(sid domain.IntegrationID, code string) (*oauth.Token, error) {

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

func (p *Provider) OAuth1Configs(sid domain.IntegrationID) (*oauthplain.Config, bool) {
	p.mx.RLock()
	defer p.mx.RUnlock()
	v, found := p.oAuth1Configs[string(sid)+"_"+p.envType]
	return v, found
}

func (p *Provider) OAuth2Configs(sid domain.IntegrationID) (*oauth.Config, bool) {
	p.mx.RLock()
	defer p.mx.RUnlock()
	v, found := p.oAuth2Configs[string(sid)+"_"+p.envType]
	return v, found
}

func (p *Provider) OAuth2Refresh(cfg *oauth.Config, token *oauth.Token) error {
	transport := &oauth.Transport{Config: cfg, Token: token}
	return transport.Refresh()
}

func (p *Provider) loadOauth2Configs(r io.Reader) error {
	p.mx.Lock()
	defer p.mx.Unlock()

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return fmt.Errorf("could not read oauth2.json, reason: %w", err)
	}
	if err := json.Unmarshal(b, &p.oAuth2Configs); err != nil {
		return fmt.Errorf("could not parse oauth2.json, reason: %w", err)
	}
	return nil
}

func (p *Provider) loadOauth1Configs(r io.Reader) error {
	p.mx.Lock()
	defer p.mx.Unlock()
	b, err := ioutil.ReadAll(r)
	if err != nil {
		return fmt.Errorf("could not read oauth1.json, reason: %w", err)
	}
	if err := json.Unmarshal(b, &p.oAuth1Configs); err != nil {
		return fmt.Errorf("could not parse oauth1.json, reason: %w", err)
	}
	return nil
}
