package main

import (
	"math/rand"
	"runtime"
	"time"

	"github.com/bugsnag/bugsnag-go"

	"github.com/toggl/pipes-api/pkg/autosync"
	"github.com/toggl/pipes-api/pkg/cfg"
	"github.com/toggl/pipes-api/pkg/pipes"
	"github.com/toggl/pipes-api/pkg/server"
	"github.com/toggl/pipes-api/pkg/storage"
	"github.com/toggl/pipes-api/pkg/toggl"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	rand.Seed(time.Now().Unix())

	flags := cfg.Flags{}
	cfg.ParseFlags(&flags)
	cfgService := cfg.NewService(flags)

	bugsnag.Configure(bugsnag.Configuration{
		APIKey:              flags.BugsnagAPIKey,
		ReleaseStage:        flags.Environment,
		NotifyReleaseStages: []string{"production", "staging"},
		// more configuration options
	})

	store := &storage.Storage{ConnString: flags.DbConnString}
	store.Connect()
	defer store.Close()

	togglService := &toggl.Service{
		URL: cfgService.GetTogglAPIHost(),
	}

	pipeService := pipes.NewPipeService(cfgService, store, togglService)

	sync := &autosync.SyncService{
		PipeService: pipeService,
	}
	sync.Start(&flags)

	c := &server.Controller{
		PipeService: pipeService,
	}

	middleware := &server.Middleware{
		PipeService: pipeService,
	}

	router := server.NewRouter(cfgService.GetCorsWhitelist())
	server.Start(&flags, router.AttachHandlers(c, middleware))
}
