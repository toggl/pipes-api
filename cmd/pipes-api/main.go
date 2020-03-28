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

	"github.com/toggl/pipes-api/pkg/config"
	"github.com/toggl/pipes-api/pkg/pipe"
	"github.com/toggl/pipes-api/pkg/pipe/autosync"
	"github.com/toggl/pipes-api/pkg/pipe/oauth"
	"github.com/toggl/pipes-api/pkg/pipe/queue"
	"github.com/toggl/pipes-api/pkg/pipe/server"
	"github.com/toggl/pipes-api/pkg/pipe/service"
	"github.com/toggl/pipes-api/pkg/pipe/storage"
	"github.com/toggl/pipes-api/pkg/toggl/client"
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

	oAuth1ConfigPath := filepath.Join(env.WorkDir, "config", "oauth1.json")
	oAuth2ConfigPath := filepath.Join(env.WorkDir, "config", "oauth2.json")
	oauthProvider := oauth.NewInMemoryProvider(env.Environment, oAuth1ConfigPath, oAuth2ConfigPath)

	togglApi := client.NewTogglApiClient(cfg.TogglAPIHost)

	pipesStore := storage.NewPostgresStorage(db)
	importsStore := storage.NewImportsPostgresStorage(db)

	integrationsConfigPath := filepath.Join(env.WorkDir, "config", "integrations.json")
	integrationsStore := storage.NewIntegrationsFileStorage(integrationsConfigPath)

	pipesQueue := queue.NewPostgresQueue(db, pipesStore)

	authorizationsStore := storage.NewAuthorizationsPostgresStorage(db)

	authFactory := &pipe.AuthorizationFactory{
		IntegrationsStorage:   integrationsStore,
		AuthorizationsStorage: authorizationsStore,
		OAuthProvider:         oauthProvider,
	}

	pipesService := service.NewService(
		oauthProvider,
		pipesStore,
		integrationsStore,
		authorizationsStore,
		importsStore,
		pipesQueue,
		togglApi,
		authFactory,
		cfg.PipesAPIHost,
	)

	autosync.NewService(pipesQueue, pipesService, env.Debug).Start()

	router := server.NewRouter(cfg.CorsWhitelist).AttachHandlers(
		server.NewController(pipesService, integrationsStore, server.Params{
			Version:   Version,
			Revision:  Revision,
			BuildTime: BuildTime,
		}),
		server.NewMiddleware(togglApi, integrationsStore),
	)
	server.Start(env.Port, router)
}
