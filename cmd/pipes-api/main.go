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

	"github.com/toggl/pipes-api/pkg/autosync"
	"github.com/toggl/pipes-api/pkg/config"
	"github.com/toggl/pipes-api/pkg/oauth"
	"github.com/toggl/pipes-api/pkg/pipe"
	"github.com/toggl/pipes-api/pkg/pipe/queue"
	"github.com/toggl/pipes-api/pkg/pipe/service"
	"github.com/toggl/pipes-api/pkg/server"
	"github.com/toggl/pipes-api/pkg/toggl/client"

	authorizationStorage "github.com/toggl/pipes-api/pkg/pipe/storage/authorization"
	idMappingStorage "github.com/toggl/pipes-api/pkg/pipe/storage/idmapping"
	importStorage "github.com/toggl/pipes-api/pkg/pipe/storage/import"
	integrationStorage "github.com/toggl/pipes-api/pkg/pipe/storage/integration"
	pipeStorage "github.com/toggl/pipes-api/pkg/pipe/storage/pipe"
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

	ps := pipeStorage.NewPostgresStorage(db)
	ims := importStorage.NewPostgresStorage(db)
	idms := idMappingStorage.NewPostgresStorage(db)

	integrationsConfigPath := filepath.Join(env.WorkDir, "config", "integrations.json")
	is := integrationStorage.NewFileStorage(integrationsConfigPath)
	as := authorizationStorage.NewPostgresStorage(db)

	qe := queue.NewPostgresQueue(db, ps)

	authFactory := &pipe.AuthorizationFactory{
		IntegrationsStorage:   is,
		AuthorizationsStorage: as,
		OAuthProvider:         oauthProvider,
	}

	pipesService := service.NewService(
		oauthProvider,
		ps,
		is,
		as,
		idms,
		ims,
		qe,
		togglApi,
		authFactory,
		cfg.PipesAPIHost,
	)

	autosync.NewService(qe, pipesService, env.Debug).Start()

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
