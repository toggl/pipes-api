package cfg

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"code.google.com/p/goauth2/oauth"
	"github.com/namsral/flag"
	"github.com/tambet/oauthplain"
)

var (
	Urls = struct {
		ReturnURL     map[string]string   `json:"return_url"`
		TogglAPIHost  map[string]string   `json:"toggl_api_host"`
		PipesAPIHost  map[string]string   `json:"pipes_api_host"`
		CorsWhitelist map[string][]string `json:"cors_whitelist"`
	}{}
	OAuth2Configs           map[string]*oauth.Config
	OAuth1Configs           map[string]*oauthplain.Config
	AvailableIntegrations   []*Integration
	AvailableAuthorizations map[string]string

	PipeType    *regexp.Regexp
	ServiceType *regexp.Regexp
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

func LoadIntegrations(flags *Flags) {
	b, err := ioutil.ReadFile(filepath.Join(flags.WorkDir, "config", "integrations.json"))
	if err != nil {
		log.Fatal(err)
	}
	if err := json.Unmarshal(b, &AvailableIntegrations); err != nil {
		log.Fatal(err)
	}
	ids := make([]string, len(AvailableIntegrations))
	for i := range AvailableIntegrations {
		ids = append(ids, AvailableIntegrations[i].ID)
	}
	ServiceType = regexp.MustCompile(strings.Join(ids, "|"))
	PipeType = regexp.MustCompile("users|projects|todolists|todos|tasks|timeentries")

	for _, integration := range AvailableIntegrations {
		AvailableAuthorizations[integration.ID] = integration.AuthType
	}
}

func LoadUrls(flags *Flags) {
	b, err := ioutil.ReadFile(filepath.Join(flags.WorkDir, "config", "urls.json"))
	if err != nil {
		log.Fatal(err)
	}
	if err := json.Unmarshal(b, &Urls); err != nil {
		log.Fatal(err)
	}
}

func LoadOauth2Configs(flags *Flags) {
	b, err := ioutil.ReadFile(filepath.Join(flags.WorkDir, "config", "oauth2.json"))
	if err != nil {
		log.Fatal(err)
	}
	if err := json.Unmarshal(b, &OAuth2Configs); err != nil {
		log.Fatal(err)
	}

}

func LoadOauth1Configs(flags *Flags) {
	b, err := ioutil.ReadFile(filepath.Join(flags.WorkDir, "config", "oauth1.json"))
	if err != nil {
		log.Fatal(err)
	}
	if err := json.Unmarshal(b, &OAuth1Configs); err != nil {
		log.Fatal(err)
	}
}
