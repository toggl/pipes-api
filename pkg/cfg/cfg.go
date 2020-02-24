package cfg

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"code.google.com/p/goauth2/oauth"
	"github.com/namsral/flag"
	"github.com/tambet/oauthplain"
)

type Flags struct {
	Port             int
	WorkDir          string
	BugsnagAPIKey    string
	Environment      string
	DbConnString     string
	TestDBConnString string
}

func ParseFlags(flags *Flags) {
	fs := flag.NewFlagSetWithEnvPrefix(os.Args[0], "PIPES_API", flag.ExitOnError)

	fs.IntVar(&flags.Port, "port", 8100, "port")
	fs.StringVar(&flags.WorkDir, "workdir", ".", "Workdir of server")
	fs.StringVar(&flags.BugsnagAPIKey, "bugsnag_key", "", "Bugsnag API Key")
	fs.StringVar(&flags.Environment, "environment", "development", "Environment")
	fs.StringVar(&flags.DbConnString, "db_conn_string", "dbname=pipes_development user=pipes_user host=localhost sslmode=disable port=5432", "DB Connection String")
	fs.StringVar(&flags.TestDBConnString, "test_db_conn_string", "dbname=pipes_test user=pipes_user host=localhost sslmode=disable port=5432", "test DB Connection String")

	fs.Parse(os.Args[1:])
}

type Service struct {
	environment string

	urls struct {
		ReturnURL     map[string]string   `json:"return_url"`
		TogglAPIHost  map[string]string   `json:"toggl_api_host"`
		PipesAPIHost  map[string]string   `json:"pipes_api_host"`
		CorsWhitelist map[string][]string `json:"cors_whitelist"`
	}

	availableIntegrations   []*Integration
	oAuth2Configs           map[string]*oauth.Config
	oAuth1Configs           map[string]*oauthplain.Config
	availableAuthorizations map[string]string
}

func NewService(flags Flags) *Service {
	svc := &Service{environment: flags.Environment}
	svc.loadUrls(flags.WorkDir)
	svc.loadIntegrations(flags.WorkDir)
	svc.loadOauth2Configs(flags.WorkDir)
	svc.loadOauth1Configs(flags.WorkDir)
	svc.fillAuthorizations(svc.availableIntegrations)
	return svc
}

func (c *Service) loadIntegrations(workDir string) {
	b, err := ioutil.ReadFile(filepath.Join(workDir, "config", "integrations.json"))
	if err != nil {
		log.Fatal(err)
	}
	if err := json.Unmarshal(b, &c.availableIntegrations); err != nil {
		log.Fatal(err)
	}
}

func (c *Service) loadOauth2Configs(workDir string) {
	b, err := ioutil.ReadFile(filepath.Join(workDir, "config", "oauth2.json"))
	if err != nil {
		log.Fatal(err)
	}
	if err := json.Unmarshal(b, &c.oAuth2Configs); err != nil {
		log.Fatal(err)
	}

}

func (c *Service) loadOauth1Configs(workDir string) {
	b, err := ioutil.ReadFile(filepath.Join(workDir, "config", "oauth1.json"))
	if err != nil {
		log.Fatal(err)
	}
	if err := json.Unmarshal(b, &c.oAuth1Configs); err != nil {
		log.Fatal(err)
	}
}

func (c *Service) fillAuthorizations(availableIntegrations []*Integration) {
	for _, integration := range availableIntegrations {
		c.availableAuthorizations[integration.ID] = integration.AuthType
	}
}

func (c *Service) GetIntegrations() []*Integration {
	return c.availableIntegrations
}

func (c *Service) loadUrls(workDir string) {
	b, err := ioutil.ReadFile(filepath.Join(workDir, "config", "urls.json"))
	if err != nil {
		log.Fatal(err)
	}
	if err := json.Unmarshal(b, &c.urls); err != nil {
		log.Fatal(err)
	}
}

func (c *Service) GetTogglAPIHost() string {
	return c.urls.TogglAPIHost[c.environment]
}

func (c *Service) GetPipesAPIHost() string {
	return c.urls.PipesAPIHost[c.environment]
}

func (c *Service) GetCorsWhitelist() []string {
	return c.urls.CorsWhitelist[c.environment]
}

func (c *Service) OAuth2URL(service string) string {
	config, ok := c.oAuth2Configs[service+"_"+c.environment]
	if !ok {
		return ""
	}
	return config.AuthCodeURL("__STATE__") + "&type=web_server"
}

func (c *Service) OAuth1Exchange(serviceID string, payload map[string]interface{}) ([]byte, error) {
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

func (c *Service) OAuth2Exchange(serviceID string, payload map[string]interface{}) ([]byte, error) {
	code := payload["code"].(string)
	if code == "" {
		return nil, errors.New("missing code")
	}
	config, res := c.oAuth2Configs[serviceID+"_"+c.environment]
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

func (c *Service) GetOAuth1Configs(serviceID string) (*oauthplain.Config, bool) {
	v, found := c.oAuth1Configs[serviceID]
	return v, found
}

func (c *Service) GetOAuth2Configs(serviceID string) (*oauth.Config, bool) {
	v, found := c.oAuth2Configs[serviceID+"_"+c.environment]
	return v, found
}

func (c *Service) GetAvailableAuthorizations(serviceID string) string {
	return c.availableAuthorizations[serviceID]
}
