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
	cfg.LoadIntegrations(&flags)
	cfg.LoadUrls(&flags)
	cfg.LoadOauth2Configs(&flags)
	cfg.LoadOauth1Configs(&flags)

	bugsnag.Configure(bugsnag.Configuration{
		APIKey:              flags.BugsnagAPIKey,
		ReleaseStage:        flags.Environment,
		NotifyReleaseStages: []string{"production", "staging"},
		// more configuration options
	})

	store := &storage.Storage{ConnString: flags.DbConnString}
	store.Connect()
	defer store.Close()

	oAuthService := &cfg.OAuthService{
		Environment: flags.Environment,
	}

	togglService := &toggl.Service{
		URL: cfg.Urls.TogglAPIHost[flags.Environment], // TODO: Remove Global state
	}

	authService := &pipes.AuthorizationService{
		Storage:                 store,
		AvailableAuthorizations: cfg.AvailableAuthorizations, // TODO: Remove Global state
		Environment:             flags.Environment,
		OAuth2Configs:           cfg.OAuth2Configs, // TODO: Remove Global state
	}

	connService := &pipes.ConnectionService{
		Storage: store,
	}

	pipeService := &pipes.PipeService{
		Storage:               store,
		AuthorizationService:  authService,
		TogglService:          togglService,
		ConnectionService:     connService,
		PipesApiHost:          cfg.Urls.PipesAPIHost[flags.Environment], // TODO: Remove Global state
		AvailableIntegrations: cfg.AvailableIntegrations,                // TODO: Remove Global state
		OAuthService:          oAuthService,
	}

	c := &server.Controller{
		Storage:              store,
		AuthorizationService: authService,
		ConnectionService:    connService,
		OAuthService:         oAuthService,
		PipeService:          pipeService,
		AvailablePipeType:    cfg.PipeType,
		AvailableServiceType: cfg.ServiceType,
	}

	sync := &autosync.SyncService{
		PipeService: pipeService,
	}
	sync.Start(&flags)

	middleware := &server.Middleware{
		TogglService:         togglService,
		AvailablePipeType:    cfg.PipeType,
		AvailableServiceType: cfg.ServiceType,
	}
	router := server.NewRouter(cfg.Urls.CorsWhitelist[flags.Environment]) // TODO: Remove Global state
	server.Start(&flags, router.AttachHandlers(c, middleware))
}
