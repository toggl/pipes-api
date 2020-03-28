package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/bugsnag/bugsnag-go"
	_ "github.com/lib/pq"

	"github.com/toggl/pipes-api/internal/autosync"
	"github.com/toggl/pipes-api/internal/config"
	"github.com/toggl/pipes-api/internal/oauth"
	"github.com/toggl/pipes-api/internal/queue"
	"github.com/toggl/pipes-api/internal/server"

	"github.com/toggl/pipes-api/pkg/domain"
	"github.com/toggl/pipes-api/pkg/toggl/client"

	"github.com/toggl/pipes-api/internal/storage"
)

var (
	Version     string
	Revision    string
	BuildTime   string
	BuildAuthor string
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	rand.Seed(time.Now().Unix())

	env := config.Flags{}
	config.ParseFlags(&env, os.Args)
	cfg := config.Load(&env)

	if env.ShowVersion {
		fmt.Printf("version: %s, revision: %s, build-time: %s, build-author: %s\n", Version, Revision, BuildTime, BuildAuthor)
		os.Exit(0)
	}

	bugsnag.Configure(bugsnag.Configuration{
		APIKey:       env.BugsnagAPIKey,
		ReleaseStage: env.Environment,
		AppVersion:   Version,
		NotifyReleaseStages: []string{
			config.EnvTypeProduction,
			config.EnvTypeStaging,
		},
		// more configuration options
	})

	db, err := sql.Open("postgres", env.DbConnString)
	if err != nil {
		log.Fatal(err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	integrationsConfig, err := os.Open(filepath.Join(env.WorkDir, "config", "integrations.json"))
	if err != nil {
		log.Fatalf("could not read integrations config file, reason: %s", err)
	}
	defer integrationsConfig.Close()

	oAuth1Config, err := os.Open(filepath.Join(env.WorkDir, "config", "oauth1.json"))
	if err != nil {
		log.Fatalf("could not read integrations config file, reason: %s", err)
	}
	defer oAuth1Config.Close()

	oAuth2Config, err := os.Open(filepath.Join(env.WorkDir, "config", "oauth2.json"))
	if err != nil {
		log.Fatalf("could not read integrations config file, reason: %s", err)
	}
	defer oAuth2Config.Close()

	oauthProvider, err := oauth.Create(env.Environment, oAuth1Config, oAuth2Config)
	if err != nil {
		log.Fatalf("couldn't create oauth provider, reason: %v", err)
	}

	togglApi := client.NewTogglApiClient(cfg.TogglAPIHost)
	ps := storage.NewPipeStorage(db)
	ims := storage.NewImportStorage(db)
	idms := storage.NewIdMappingStorageStorage(db)
	is := storage.NewIntegrationStorage(integrationsConfig)
	as := storage.NewAuthorizationStorage(db)

	authFactory := &domain.AuthorizationFactory{
		IntegrationsStorage:   is,
		AuthorizationsStorage: as,
		OAuthProvider:         oauthProvider,
	}

	pipeFactory := &domain.PipeFactory{
		AuthorizationFactory:  authFactory,
		AuthorizationsStorage: as,
		PipesStorage:          ps,
		ImportsStorage:        ims,
		IDMappingsStorage:     idms,
		TogglClient:           togglApi,
	}

	qe := queue.NewPipesQueue(db, pipeFactory, ps)

	pipesService := &domain.Service{
		AuthorizationFactory:  authFactory,
		PipeFactory:           pipeFactory,
		PipesStorage:          ps,
		AuthorizationsStorage: as,
		IntegrationsStorage:   is,
		IDMappingsStorage:     idms,
		ImportsStorage:        ims,
		OAuthProvider:         oauthProvider,
		TogglClient:           togglApi,
		Queue:                 qe,
	}

	autosync.NewService(qe, env.Debug).Start()

	router := server.NewRouter(cfg.CorsWhitelist).AttachHandlers(
		server.NewController(pipesService, is, server.Params{
			Version:   Version,
			Revision:  Revision,
			BuildTime: BuildTime,
		}),
		server.NewMiddleware(togglApi, is),
	)
	server.Start(env.Port, router)
}
