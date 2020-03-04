package pipe

import (
	"database/sql"
	"encoding/json"
	"log"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	goauth2 "code.google.com/p/goauth2/oauth"
	"github.com/stretchr/testify/suite"

	"github.com/toggl/pipes-api/pkg/config"
	"github.com/toggl/pipes-api/pkg/integrations"
	"github.com/toggl/pipes-api/pkg/oauth"
	"github.com/toggl/pipes-api/pkg/toggl/client"
)

var (
	workspaceID                                = 1
	pipeID      integrations.PipeID            = "users"
	serviceID   integrations.ExternalServiceID = "basecamp"
)

func TestNewClient(t *testing.T) {
	t.Skipf("DEPRECATED TEST: Should be removed after new will be created")
	expectedKey := "basecamp:users"
	p := NewPipe(workspaceID, serviceID, pipeID)

	if p.Key != expectedKey {
		t.Errorf("NewPipe Key = %v, want %v", p.Key, expectedKey)
	}
}

func TestGetPipesFromQueue_DoesNotReturnMultipleSameWorkspace(t *testing.T) {
	t.Skipf("DEPRECATED TEST: Should be removed after new will be created")

	flags := config.Flags{}
	config.ParseFlags(&flags)
	cfg := config.Load(&flags)

	db, err := sql.Open("postgres", flags.DbConnString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	api := client.NewTogglApiClient(cfg.TogglAPIHost)

	oAuth1ConfigPath := filepath.Join(cfg.WorkDir, "config", "oauth1.json")
	oAuth2ConfigPath := filepath.Join(cfg.WorkDir, "config", "oauth2.json")
	oauthProvider := oauth.NewProvider(cfg.EnvType, oAuth1ConfigPath, oAuth2ConfigPath)

	pipesStorage := NewStorage(db)
	pipeService := NewService(oauthProvider, pipesStorage, api, cfg.PipesAPIHost, cfg.WorkDir)

	createAndEnqueuePipeFn := func(workspaceID int, serviceID integrations.ExternalServiceID, pipeID integrations.PipeID, priority int) *Pipe {
		pipe := NewPipe(workspaceID, serviceID, pipeID)
		pipe.Automatic = true
		pipe.Configured = true
		data, err := json.Marshal(pipe)
		if err != nil {
			t.Error(err)
			return nil
		}
		_, err = db.Exec(`
			with created as (
				insert into store(workspace_id, Key, data)
				values ($1, $2, $3)
				returning *
			)
			insert into queued_pipes(workspace_id, Key, priority)
			select created.workspace_id, created.Key, $4 from created
		`, pipe.WorkspaceID, pipe.Key, data, priority)
		if err != nil {
			t.Error(err)
		}
		return pipe
	}

	createAndEnqueuePipeFn(1, "asana", "users", 0)
	createAndEnqueuePipeFn(2, "asana", "projects", 10)
	createAndEnqueuePipeFn(1, "asana", "projects", 0)
	createAndEnqueuePipeFn(3, "asana", "projects", 100)

	// first fetch should return 3 store and unique per workspace
	pipes, err := pipeService.GetPipesFromQueue()
	if err != nil {
		t.Error(err)
	}
	if len(pipes) != 3 {
		t.Error("should return 3 store")
	}
	if pipes[0].WorkspaceID != 3 {
		t.Error("first returned pipe should be pipe for workspace 3 because it has highest priority")
	}

	// make sure returned store are unique per workspace
	retrievedWorkspace := map[int]bool{}
	for _, pipe := range pipes {
		_, exists := retrievedWorkspace[pipe.WorkspaceID]
		if exists {
			t.Error("there's already existing queued pipe with workspace id ", pipe.WorkspaceID)
		}

		retrievedWorkspace[pipe.WorkspaceID] = true
		err = pipeService.SetQueuedPipeSynced(pipe)
		if err != nil {
			t.Error(err)
		}
	}

	// second fetch should return 1 pipe left
	pipes, err = pipeService.GetPipesFromQueue()
	if err != nil {
		t.Error(err)
	}
	if len(pipes) != 1 {
		t.Error("should only return 1 pipe")
	}
	if pipes[0].WorkspaceID != 1 {
		t.Error("should be workspace 1")
	}
	if _, exists := retrievedWorkspace[pipes[0].WorkspaceID]; !exists {
		t.Error("should return pipe with workspace from retrievedWorkspace")
	}
}

func TestWorkspaceIntegrations(t *testing.T) {
	t.Skipf("DEPRECATED TEST: Should be removed after new will be created")

	flags := config.Flags{}
	config.ParseFlags(&flags)

	cfg := config.Load(&flags)

	db, err := sql.Open("postgres", flags.DbConnString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	api := client.NewTogglApiClient(cfg.TogglAPIHost)

	oAuth1ConfigPath := filepath.Join(cfg.WorkDir, "config", "oauth1.json")
	oAuth2ConfigPath := filepath.Join(cfg.WorkDir, "config", "oauth2.json")
	oauthProvider := oauth.NewProvider(cfg.EnvType, oAuth1ConfigPath, oAuth2ConfigPath)

	pipesStorage := NewStorage(db)
	pipeService := NewService(oauthProvider, pipesStorage, api, cfg.PipesAPIHost, cfg.WorkDir)

	integrations, err := pipeService.WorkspaceIntegrations(workspaceID)

	if err != nil {
		t.Fatalf("workspaceIntegrations returned error: %v", err)
	}

	for i := range integrations {
		integrations[i].Pipes = nil
	}

	want := []Integration{
		{ID: "basecamp", Name: "Basecamp", Link: "https://support.toggl.com/import-and-export/integrations-via-toggl-store/integration-with-basecamp", Image: "/images/logo-basecamp.png", AuthType: "oauth2"},
		{ID: "freshbooks", Name: "Freshbooks", Link: "https://support.toggl.com/import-and-export/integrations-via-toggl-store/integration-with-freshbooks-classic", Image: "/images/logo-freshbooks.png", AuthType: "oauth1"},
		{ID: "teamweek", Name: "Toggl Plan", Link: "https://support.toggl.com/en/articles/2212490-integration-with-toggl-plan-teamweek", Image: "/images/logo-teamweek.png", AuthType: "oauth2"},
		{ID: "asana", Name: "Asana", Link: "https://support.toggl.com/import-and-export/integrations-via-toggl-store/integration-with-asana", Image: "/images/logo-asana.png", AuthType: "oauth2"},
		{ID: "github", Name: "Github", Link: "https://support.toggl.com/import-and-export/integrations-via-toggl-store/integration-with-github", Image: "/images/logo-github.png", AuthType: "oauth2"},
	}

	if len(integrations) != len(want) {
		t.Fatalf("Load integration(s) detected - please add tests!")
	}

	for i := range integrations {
		if !reflect.DeepEqual(integrations[i], want[i]) {
			t.Fatalf("workspaceIntegrations returned  ---------->\n%+v, \nwant ---------->\n%+v", integrations[i], want[i])
		}
	}
}

func TestWorkspaceIntegrationPipes(t *testing.T) {
	t.Skipf("DEPRECATED TEST: Should be removed after new will be created")

	flags := config.Flags{}
	config.ParseFlags(&flags)
	cfg := config.Load(&flags)

	db, err := sql.Open("postgres", flags.DbConnString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	api := client.NewTogglApiClient(cfg.TogglAPIHost)

	oAuth1ConfigPath := filepath.Join(cfg.WorkDir, "config", "oauth1.json")
	oAuth2ConfigPath := filepath.Join(cfg.WorkDir, "config", "oauth2.json")
	oauthProvider := oauth.NewProvider(cfg.EnvType, oAuth1ConfigPath, oAuth2ConfigPath)

	pipesStorage := NewStorage(db)
	pipeService := NewService(oauthProvider, pipesStorage, api, cfg.PipesAPIHost, cfg.WorkDir)

	integrations, err := pipeService.WorkspaceIntegrations(workspaceID)

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
		t.Fatalf("Load integration(s) detected - please add tests!")
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

type ServiceTestSuite struct {
	suite.Suite
	db *sql.DB
}

func (ts *ServiceTestSuite) TestService_Refresh_Load_Ok() {
	ts.T().Skipf("TODO: Fix, not working because of configs")
	flags := config.Flags{}
	config.ParseFlags(&flags)
	cfg := config.Load(&flags)

	integrationsConfigPath := filepath.Join(cfg.WorkDir, "config", "integrations.json")

	s := NewStorage(ts.db)
	api := client.NewTogglApiClient("https://localhost")
	sb := &stubOauthProvider{}

	svc := NewService(sb, s, api, "https://localhost", integrationsConfigPath)

	svc.setAuthorizationType("github", TypeOauth2)

	a1 := NewAuthorization(1, "github")
	t := goauth2.Token{
		AccessToken:  "123",
		RefreshToken: "456",
		Expiry:       time.Now().Add(-time.Hour),
		Extra:        nil,
	}
	b, err := json.Marshal(t)
	ts.NoError(err)
	a1.Data = b

	err = svc.refreshAuthorization(a1)
	ts.NoError(err)

	aSaved, err := s.LoadAuthorization(1, "github")
	ts.NoError(err)
	ts.NotEqual([]byte("{}"), aSaved.Data)
}

func (ts *ServiceTestSuite) TestService_Refresh_Oauth1() {
	ts.T().Skipf("TODO: Fix, not working because of configs")

	s := NewStorage(ts.db)
	api := client.NewTogglApiClient("https://localhost")
	sb := &stubOauthProvider{}

	svc := NewService(sb, s, api, "", "")

	svc.setAuthorizationType("github", TypeOauth1)

	a1 := NewAuthorization(1, "asana")

	err := svc.refreshAuthorization(a1)
	ts.NoError(err)
}

func (ts *ServiceTestSuite) TestService_Refresh_NotExpired() {
	ts.T().Skipf("TODO: Fix, not working because of configs")

	s := NewStorage(ts.db)
	api := client.NewTogglApiClient("https://localhost")
	sb := &stubOauthProvider{}

	svc := NewService(sb, s, api, "", "")
	svc.setAuthorizationType("github", TypeOauth2)

	a1 := NewAuthorization(1, "github")
	t := goauth2.Token{
		AccessToken:  "123",
		RefreshToken: "456",
		Expiry:       time.Now().Add(time.Hour * 24),
		Extra:        nil,
	}
	b, err := json.Marshal(t)
	ts.NoError(err)
	a1.Data = b

	err = svc.refreshAuthorization(a1)
	ts.NoError(err)
}

func (ts *ServiceTestSuite) TestService_Set_GetAvailableAuthorizations() {
	ts.T().Skipf("TODO: Fix, not working because of configs")

	s := NewStorage(ts.db)
	api := client.NewTogglApiClient("https://localhost")
	sb := &stubOauthProvider{}

	svc := NewService(sb, s, api, "", "")

	res := svc.getAvailableAuthorizations("github")
	ts.Equal("", res)

	svc.setAuthorizationType("github", TypeOauth2)
	svc.setAuthorizationType("asana", TypeOauth1)

	res = svc.getAvailableAuthorizations("github")
	ts.Equal(TypeOauth2, res)

	res = svc.getAvailableAuthorizations("asana")
	ts.Equal(TypeOauth1, res)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ServiceTestSuite))
}
