package pipes

import (
	"reflect"
	"testing"

	"github.com/toggl/pipes-api/pkg/cfg"
	"github.com/toggl/pipes-api/pkg/storage"
)

var testDB = "pipes_test"

func TestWorkspaceIntegrations(t *testing.T) {
	flags := cfg.Flags{}
	cfg.ParseFlags(&flags)

	oauthService := &cfg.OAuthService{
		Environment: flags.Environment,
	}

	store := &storage.Storage{
		ConnString: flags.TestDBConnString,
	}

	authService := &AuthorizationService{
		Storage:                 store,
		AvailableAuthorizations: cfg.AvailableAuthorizations, // TODO: Remove global state
		Environment:             flags.Environment,
		OAuth2Configs:           cfg.OAuth2Configs, // TODO: Remove global state
	}

	integrationSvc := IntegrationService{
		AuthorizationService:  authService,
		AvailableIntegrations: cfg.AvailableIntegrations, // TODO: Remove global state
		OAuthService:          oauthService,
	}

	integrations, err := integrationSvc.WorkspaceIntegrations(workspaceID)

	if err != nil {
		t.Fatalf("workspaceIntegrations returned error: %v", err)
	}

	for i := range integrations {
		integrations[i].Pipes = nil
	}

	want := []Integration{
		{ID: "basecamp", Name: "Basecamp", Link: "https://support.toggl.com/import-and-export/integrations-via-toggl-pipes/integration-with-basecamp", Image: "/images/logo-basecamp.png", AuthType: "oauth2"},
		{ID: "freshbooks", Name: "Freshbooks", Link: "https://support.toggl.com/import-and-export/integrations-via-toggl-pipes/integration-with-freshbooks-classic", Image: "/images/logo-freshbooks.png", AuthType: "oauth1"},
		{ID: "teamweek", Name: "Toggl Plan", Link: "https://support.toggl.com/articles/2212490-integration-with-toggl-plan-teamweek", Image: "/images/logo-teamweek.png", AuthType: "oauth2"},
		{ID: "asana", Name: "Asana", Link: "https://support.toggl.com/import-and-export/integrations-via-toggl-pipes/integration-with-asana", Image: "/images/logo-asana.png", AuthType: "oauth2"},
		{ID: "github", Name: "Github", Link: "https://support.toggl.com/import-and-export/integrations-via-toggl-pipes/integration-with-github", Image: "/images/logo-github.png", AuthType: "oauth2"},
	}

	if len(integrations) != len(want) {
		t.Fatalf("New integration(s) detected - please add tests!")
	}

	for i := range integrations {
		if !reflect.DeepEqual(integrations[i], want[i]) {
			t.Fatalf("workspaceIntegrations returned  ---------->\n%+v, \nwant ---------->\n%+v", integrations[i], want[i])
		}
	}
}

func TestWorkspaceIntegrationPipes(t *testing.T) {

	flags := cfg.Flags{}
	cfg.ParseFlags(&flags)

	oauthService := &cfg.OAuthService{
		Environment: flags.Environment,
	}

	store := &storage.Storage{
		ConnString: flags.TestDBConnString,
	}

	authService := &AuthorizationService{
		Storage:                 store,
		AvailableAuthorizations: cfg.AvailableAuthorizations, // TODO: Remove global state
		Environment:             flags.Environment,
		OAuth2Configs:           cfg.OAuth2Configs, // TODO: Remove global state
	}

	integrationSvc := IntegrationService{
		AuthorizationService:  authService,
		AvailableIntegrations: cfg.AvailableIntegrations, // TODO: Remove global state
		OAuthService:          oauthService,
	}

	integrations, err := integrationSvc.WorkspaceIntegrations(workspaceID)

	if err != nil {
		t.Fatalf("workspaceIntegrations returned error: %v", err)
	}

	want := [][]*Pipe{
		{ // Basecamp
			{ID: "users", Name: "Users", Premium: false, AutomaticOption: false},
			{ID: "projects", Name: "Projects", Premium: false, AutomaticOption: true},
			{ID: "todolists", Name: "Todo lists", Premium: true, AutomaticOption: true},
			{ID: "todos", Name: "Todos", Premium: true, AutomaticOption: true},
		},
		{ // Freshbooks
			{ID: "users", Name: "Users", Premium: false, AutomaticOption: false},
			{ID: "projects", Name: "Projects", Premium: false, AutomaticOption: true},
			{ID: "tasks", Name: "Tasks", Premium: true, AutomaticOption: true},
			{ID: "timeentries", Name: "Time entries", Premium: true, AutomaticOption: true},
		},
		{ // Teamweek
			{ID: "users", Name: "Users", Premium: false, AutomaticOption: false},
			{ID: "projects", Name: "Projects", Premium: false, AutomaticOption: true},
			{ID: "tasks", Name: "Tasks", Premium: true, AutomaticOption: true},
		},
		{ // Asana
			{ID: "users", Name: "Users", Premium: false, AutomaticOption: false},
			{ID: "projects", Name: "Projects", Premium: false, AutomaticOption: true},
			{ID: "tasks", Name: "Tasks", Premium: true, AutomaticOption: true},
		},
		{ // Github
			{ID: "projects", Name: "Github repos", Premium: false, AutomaticOption: true},
		},
	}

	if len(integrations) != len(want) {
		t.Fatalf("New integration(s) detected - please add tests!")
	}

	for i := range integrations {
		for j, pipe := range integrations[i].Pipes {
			pipe.Description = ""
			if !reflect.DeepEqual(pipe, want[i][j]) {
				t.Fatalf("workspaceIntegrations returned  ---------->\n%+v, \nwant ---------->\n%+v", pipe, want[i][j])
			}
		}
	}
}
