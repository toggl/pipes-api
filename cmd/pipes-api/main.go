package main

import (
	"database/sql"
	"log"
	"math/rand"
	"runtime"
	"time"

	"github.com/bugsnag/bugsnag-go"

	"github.com/toggl/pipes-api/pkg/autosync"
	"github.com/toggl/pipes-api/pkg/environment"
	"github.com/toggl/pipes-api/pkg/server"
	"github.com/toggl/pipes-api/pkg/storage"
	"github.com/toggl/pipes-api/pkg/toggl"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	rand.Seed(time.Now().Unix())

	envFlags := environment.Flags{}
	environment.ParseFlags(&envFlags)

	env := environment.New(
		envFlags.Environment,
		envFlags.WorkDir,
	)

	bugsnag.Configure(bugsnag.Configuration{
		APIKey:       envFlags.BugsnagAPIKey,
		ReleaseStage: envFlags.Environment,
		NotifyReleaseStages: []string{
			environment.EnvTypeProduction,
			environment.EnvTypeStaging,
		},
		// more configuration options
	})

	db, err := sql.Open("postgres", envFlags.DbConnString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	api := toggl.NewApiClient(env.GetTogglAPIHost())
	pipes := storage.NewPipesStorage(env, api, db)

	autosync.NewService(envFlags.Environment, pipes).Start()

	router := server.NewRouter(env.GetCorsWhitelist()).AttachHandlers(
		server.NewController(env, pipes, api),
		server.NewMiddleware(api, pipes, pipes),
	)
	server.Start(envFlags.Port, router)
}
