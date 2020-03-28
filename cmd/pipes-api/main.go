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

	authorizationStorage "github.com/toggl/pipes-api/internal/storage/authorization"
	idMappingStorage "github.com/toggl/pipes-api/internal/storage/idmapping"
	importStorage "github.com/toggl/pipes-api/internal/storage/import"
	integrationStorage "github.com/toggl/pipes-api/internal/storage/integration"
	pipeStorage "github.com/toggl/pipes-api/internal/storage/pipe"
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

	qe := queue.NewPostgresQueue(db, pipeFactory, ps)

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
