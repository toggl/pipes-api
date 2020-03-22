package main

import (
	"database/sql"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/bugsnag/bugsnag-go"
	_ "github.com/lib/pq"

	"github.com/toggl/pipes-api/pkg/config"
	"github.com/toggl/pipes-api/pkg/oauth"
	"github.com/toggl/pipes-api/pkg/pipe/autosync"
	"github.com/toggl/pipes-api/pkg/pipe/queue"
	"github.com/toggl/pipes-api/pkg/pipe/server"
	"github.com/toggl/pipes-api/pkg/pipe/service"
	"github.com/toggl/pipes-api/pkg/pipe/storage"
	"github.com/toggl/pipes-api/pkg/toggl/client"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	rand.Seed(time.Now().Unix())

	env := config.Flags{}
	config.ParseFlags(&env, os.Args)
	cfg := config.Load(&env)

	bugsnag.Configure(bugsnag.Configuration{
		APIKey:       env.BugsnagAPIKey,
		ReleaseStage: env.Environment,
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

	api := client.NewTogglApiClient(cfg.TogglAPIHost)

	integrationsConfigPath := filepath.Join(env.WorkDir, "config", "integrations.json")
	pipesStore := storage.NewPostgresStorage(db)
	pipesStore.LoadIntegrationsFromConfig(integrationsConfigPath)

	pipesQueue := queue.NewPostgresQueue(db, pipesStore)

	pipesService := service.NewService(oauthProvider, pipesStore, pipesQueue, api, cfg.PipesAPIHost)

	autosync.NewService(pipesQueue, pipesService).Start()

	router := server.NewRouter(cfg.CorsWhitelist).AttachHandlers(
		server.NewController(pipesService, pipesStore),
		server.NewMiddleware(api, pipesStore),
	)
	server.Start(env.Port, router)
}
