package main

import (
	"database/sql"
	"log"
	"math/rand"
	"runtime"
	"time"

	"github.com/toggl/pipes-api/pkg/autosync"
	"github.com/toggl/pipes-api/pkg/environment"
	"github.com/toggl/pipes-api/pkg/errnotifier"
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

	bugSnagNotifier := errnotifier.NewBugSnagNotifier(
		envFlags.BugsnagAPIKey,
		envFlags.Environment,
	)

	db, err := sql.Open("postgres", envFlags.DbConnString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	api := toggl.NewApiClient(env.GetTogglAPIHost())
	pipes := storage.NewPipesStorage(env, api, db, bugSnagNotifier)

	autosync.NewService(envFlags.Environment, pipes, bugSnagNotifier).Start()

	router := server.NewRouter(env.GetCorsWhitelist()).AttachHandlers(
		server.NewController(env, pipes),
		server.NewMiddleware(api, pipes, pipes),
	)
	server.Start(envFlags.Port, router)
}
