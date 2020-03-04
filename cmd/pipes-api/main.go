package main

import (
	"database/sql"
	"log"
	"math/rand"
	"path/filepath"
	"runtime"
	"time"

	"github.com/bugsnag/bugsnag-go"
	_ "github.com/lib/pq"

	"github.com/toggl/pipes-api/pkg/autosync"
	"github.com/toggl/pipes-api/pkg/config"
	"github.com/toggl/pipes-api/pkg/connection"
	"github.com/toggl/pipes-api/pkg/oauth"
	"github.com/toggl/pipes-api/pkg/pipe"
	"github.com/toggl/pipes-api/pkg/server"
	"github.com/toggl/pipes-api/pkg/toggl/client"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	rand.Seed(time.Now().Unix())

	env := config.Flags{}
	config.ParseFlags(&env)
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
	defer db.Close()

	oAuth1ConfigPath := filepath.Join(env.WorkDir, "config", "oauth1.json")
	oAuth2ConfigPath := filepath.Join(env.WorkDir, "config", "oauth2.json")
	oauthProvider := oauth.NewProvider(env.Environment, oAuth1ConfigPath, oAuth2ConfigPath)

	api := client.NewTogglApiClient(cfg.TogglAPIHost)

	connStore := connection.NewStorage(db)
	pipesStore := pipe.NewStorage(db)

	integrationsConfigPath := filepath.Join(env.WorkDir, "config", "integrations.json")
	pipesService := pipe.NewService(oauthProvider, pipesStore, connStore, api, cfg.PipesAPIHost, integrationsConfigPath)

	autosync.NewService(pipesService).Start()

	router := server.NewRouter(cfg.CorsWhitelist).AttachHandlers(
		server.NewController(pipesService),
		server.NewMiddleware(api, pipesService),
	)
	server.Start(env.Port, router)
}
