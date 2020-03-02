package main

import (
	"database/sql"
	"log"
	"math/rand"
	"runtime"
	"time"

	"github.com/bugsnag/bugsnag-go"
	_ "github.com/lib/pq"

	"github.com/toggl/pipes-api/pkg/authorization"
	"github.com/toggl/pipes-api/pkg/autosync"
	"github.com/toggl/pipes-api/pkg/connection"
	"github.com/toggl/pipes-api/pkg/environment"
	"github.com/toggl/pipes-api/pkg/pipes"
	"github.com/toggl/pipes-api/pkg/server"
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
	authStore := authorization.NewStorage(db, env)
	connStore := connection.NewStorage(db)
	pipesStore := pipes.NewStorage(env, db)
	pipesService := pipes.NewService(env, authStore, pipesStore, connStore, api)

	autosync.NewService(envFlags.Environment, pipesService).Start()

	router := server.NewRouter(env.GetCorsWhitelist()).AttachHandlers(
		server.NewController(env, pipesService, pipesStore, authStore, api),
		server.NewMiddleware(api, pipesService, pipesService),
	)
	server.Start(envFlags.Port, router)
}
