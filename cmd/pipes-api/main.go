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

	"github.com/toggl/pipes-api/internal/config"
	"github.com/toggl/pipes-api/internal/oauth"
	"github.com/toggl/pipes-api/internal/server"
	"github.com/toggl/pipes-api/internal/service"
	"github.com/toggl/pipes-api/internal/storage"
	"github.com/toggl/pipes-api/internal/sync"
	"github.com/toggl/pipes-api/internal/toggl"
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

	oauthProvider, err := oauth.NewProvider(env.Environment, oAuth1Config, oAuth2Config)
	if err != nil {
		log.Fatalf("couldn't create oauth provider, reason: %v", err)
	}

	apiClient := toggl.NewApiClient(cfg.TogglAPIHost)
	pipeStorage := storage.NewPipeStorage(db)
	importStorage := storage.NewImportStorage(db)
	idMappingStorage := storage.NewIdMappingStorage(db)
	authorizationStorage := storage.NewAuthorizationStorage(db)
	integrationStorage := storage.NewIntegrationStorage(integrationsConfig)

	pipeService := service.NewPipeService(pipeStorage)

	pipeSyncService := service.NewPipeSyncService(
		pipeStorage,
		authorizationStorage,
		integrationStorage,
		idMappingStorage,
		importStorage,
		oauthProvider,
		apiClient,
	)

	healthCheckService := service.NewHealthCheckService(
		pipeStorage,
		apiClient,
	)

	authorizationService := service.NewAuthorizationService(
		pipeStorage,
		authorizationStorage,
		integrationStorage,
		oauthProvider,
	)

	pipeQueue := sync.NewQueue(db, pipeSyncService, pipeStorage)

	workerPool := sync.NewWorkerPool(pipeQueue, pipeSyncService, env.Debug)
	workerPool.Start()

	controller := server.NewController(
		pipeService,
		pipeSyncService,
		healthCheckService,
		authorizationService,
		integrationStorage,
		pipeQueue,
		server.Params{
			Version:   Version,
			Revision:  Revision,
			BuildTime: BuildTime,
		})

	middleware := server.NewMiddleware(integrationStorage, apiClient)

	router := server.NewRouter(cfg.CorsWhitelist).AttachHandlers(controller, middleware)
	server.Start(env.Port, router)
}
