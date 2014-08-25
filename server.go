package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"code.google.com/p/goauth2/oauth"
	"github.com/toggl/bugsnag"
	"github.com/toggl/oauthplain"
)

var (
	urls = struct {
		ReturnURL     map[string]string   `json:"return_url"`
		TogglAPIHost  map[string]string   `json:"toggl_api_host"`
		PipesAPIHost  map[string]string   `json:"pipes_api_host"`
		CorsWhitelist map[string][]string `json:"cors_whitelist"`
	}{}
	availableAuthorizations = map[string]string{}
	oAuth2Configs           map[string]*oauth.Config
	oAuth1Configs           map[string]*oauthplain.Config
	availableIntegrations   []*Integration

	pipeType    *regexp.Regexp
	serviceType *regexp.Regexp
)

func main() {
	flag.Parse()
	runtime.GOMAXPROCS(runtime.NumCPU())

	bugsnag.Verbose = true
	bugsnag.APIKey = *bugsnagAPIKey
	bugsnag.ReleaseStage = *environment
	bugsnag.NotifyReleaseStages = []string{"staging", "production"}

	db = connectDB(*dbHost, *dbPort, *dbName, *dbUser, *dbPass)
	defer db.Close()

	loadIntegrations()

	b, err := ioutil.ReadFile(filepath.Join(*workdir, "config", "urls.json"))
	if err != nil {
		log.Fatal(err)
	}
	if err := json.Unmarshal(b, &urls); err != nil {
		log.Fatal(err)
	}
	b, err = ioutil.ReadFile(filepath.Join(*workdir, "config", "oauth2.json"))
	if err != nil {
		log.Fatal(err)
	}
	if err := json.Unmarshal(b, &oAuth2Configs); err != nil {
		log.Fatal(err)
	}
	b, err = ioutil.ReadFile(filepath.Join(*workdir, "config", "oauth1.json"))
	if err != nil {
		log.Fatal(err)
	}
	if err := json.Unmarshal(b, &oAuth1Configs); err != nil {
		log.Fatal(err)
	}

	for _, integration := range availableIntegrations {
		availableAuthorizations[integration.ID] = integration.AuthType
	}

	if *environment == "production" {
		go autoSyncRunner()
	}

	log.Println(fmt.Sprintf("=> Starting in %s on http://0.0.0.0:%d", *environment, *port))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), http.DefaultServeMux))
}

func loadIntegrations() {
	b, err := ioutil.ReadFile(filepath.Join(*workdir, "config", "integrations.json"))
	if err != nil {
		log.Fatal(err)
	}
	if err := json.Unmarshal(b, &availableIntegrations); err != nil {
		log.Fatal(err)
	}
	ids := make([]string, len(availableIntegrations))
	for i := range availableIntegrations {
		ids = append(ids, availableIntegrations[i].ID)
	}
	serviceType = regexp.MustCompile(strings.Join(ids, "|"))
	pipeType = regexp.MustCompile("users|projects|todolists|todos|tasks|timeentries")
}

func isWhiteListedCorsOrigin(r *http.Request) (string, bool) {
	origin := r.Header.Get("Origin")
	if allowedDomains, exist := urls.CorsWhitelist[*environment]; exist {
		for _, s := range allowedDomains {
			if s == origin {
				return origin, true
			}
		}
	}
	return "", false
}
