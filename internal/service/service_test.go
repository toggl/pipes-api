package service

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/toggl/pipes-api/internal/config"
	"github.com/toggl/pipes-api/internal/oauth"
	"github.com/toggl/pipes-api/internal/queue"

	"github.com/toggl/pipes-api/pkg/domain"
	"github.com/toggl/pipes-api/pkg/domain/mocks"
	"github.com/toggl/pipes-api/pkg/integration"
	"github.com/toggl/pipes-api/pkg/toggl/client"

	authorizationStorage "github.com/toggl/pipes-api/internal/storage/authorization"
	idMappingStorage "github.com/toggl/pipes-api/internal/storage/idmapping"
	importStorage "github.com/toggl/pipes-api/internal/storage/import"
	integrationStorage "github.com/toggl/pipes-api/internal/storage/integration"
	pipeStorage "github.com/toggl/pipes-api/internal/storage/pipe"
)

var workspaceID = 1

func TestGetPipesFromQueue_DoesNotReturnMultipleSameWorkspace(t *testing.T) {
	t.Skipf("DEPRECATED TEST: Should be removed after new will be created")

	flags := config.Flags{}
	config.ParseFlags(&flags, os.Args)
	cfg := config.Load(&flags)

	db, err := sql.Open("postgres", flags.DbConnString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	api := client.NewTogglApiClient(cfg.TogglAPIHost)

	oAuth1ConfigPath := filepath.Join(cfg.WorkDir, "config", "oauth1.json")
	oAuth2ConfigPath := filepath.Join(cfg.WorkDir, "config", "oauth2.json")
	oauthProvider := oauth.NewInMemoryProvider(cfg.EnvType, oAuth1ConfigPath, oAuth2ConfigPath)

	importsStorage := importStorage.NewPostgresStorage(db)
	pipesStorage := pipeStorage.NewPostgresStorage(db)
	integrationsConfigPath := filepath.Join(cfg.WorkDir, "config", "integrations.json")
	integrationsStorage := integrationStorage.NewFileStorage(integrationsConfigPath)
	authorizationsStorage := authorizationStorage.NewPostgresStorage(db)
	pipesQueue := queue.NewPostgresQueue(db, pipesStorage)
	idMappingsStore := idMappingStorage.NewPostgresStorage(db)

	authFactory := &domain.AuthorizationFactory{
		IntegrationsStorage:   integrationsStorage,
		AuthorizationsStorage: authorizationsStorage,
		OAuthProvider:         oauthProvider,
	}

	pipeService := NewService(
		oauthProvider,
		pipesStorage,
		integrationsStorage,
		authorizationsStorage,
		idMappingsStore,
		importsStorage,
		pipesQueue,
		api,
		authFactory,
		cfg.PipesAPIHost,
	)

	createAndEnqueuePipeFn := func(workspaceID int, serviceID integration.ID, pipeID integration.PipeID, priority int) *domain.Pipe {
		pipe := domain.NewPipe(workspaceID, serviceID, pipeID)
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
	pipes, err := pipeService.queue.GetPipesFromQueue()
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
		err = pipeService.queue.SetQueuedPipeSynced(pipe)
		if err != nil {
			t.Error(err)
		}
	}

	// second fetch should return 1 pipe left
	pipes, err = pipeService.queue.GetPipesFromQueue()
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
	config.ParseFlags(&flags, os.Args)

	cfg := config.Load(&flags)

	db, err := sql.Open("postgres", flags.DbConnString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	api := client.NewTogglApiClient(cfg.TogglAPIHost)

	oAuth1ConfigPath := filepath.Join(cfg.WorkDir, "config", "oauth1.json")
	oAuth2ConfigPath := filepath.Join(cfg.WorkDir, "config", "oauth2.json")
	oauthProvider := oauth.NewInMemoryProvider(cfg.EnvType, oAuth1ConfigPath, oAuth2ConfigPath)

	importsStorage := importStorage.NewPostgresStorage(db)
	pipesStorage := pipeStorage.NewPostgresStorage(db)
	integrationsConfigPath := filepath.Join(cfg.WorkDir, "config", "integrations.json")
	integrationsStorage := integrationStorage.NewFileStorage(integrationsConfigPath)
	authorizationsStorage := authorizationStorage.NewPostgresStorage(db)
	idMappingsStore := idMappingStorage.NewPostgresStorage(db)

	pipesQueue := queue.NewPostgresQueue(db, pipesStorage)
	authFactory := &domain.AuthorizationFactory{
		IntegrationsStorage:   integrationsStorage,
		AuthorizationsStorage: authorizationsStorage,
		OAuthProvider:         oauthProvider,
	}
	pipeService := NewService(
		oauthProvider,
		pipesStorage,
		integrationsStorage,
		authorizationsStorage,
		idMappingsStore,
		importsStorage,
		pipesQueue,
		api,
		authFactory,
		cfg.PipesAPIHost,
	)

	integrations, err := pipeService.GetIntegrations(workspaceID)

	if err != nil {
		t.Fatalf("workspaceIntegrations returned error: %v", err)
	}

	for i := range integrations {
		integrations[i].Pipes = nil
	}

	want := []domain.Integration{
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
	config.ParseFlags(&flags, os.Args)
	cfg := config.Load(&flags)

	db, err := sql.Open("postgres", flags.DbConnString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	api := client.NewTogglApiClient(cfg.TogglAPIHost)

	oAuth1ConfigPath := filepath.Join(cfg.WorkDir, "config", "oauth1.json")
	oAuth2ConfigPath := filepath.Join(cfg.WorkDir, "config", "oauth2.json")
	oauthProvider := oauth.NewInMemoryProvider(cfg.EnvType, oAuth1ConfigPath, oAuth2ConfigPath)

	importsStorage := importStorage.NewPostgresStorage(db)
	pipesStorage := pipeStorage.NewPostgresStorage(db)
	integrationsConfigPath := filepath.Join(cfg.WorkDir, "config", "integrations.json")
	integrationsStorage := integrationStorage.NewFileStorage(integrationsConfigPath)
	pipesQueue := queue.NewPostgresQueue(db, pipesStorage)
	authorizationsStorage := authorizationStorage.NewPostgresStorage(db)
	idMappingsStore := idMappingStorage.NewPostgresStorage(db)

	authFactory := &domain.AuthorizationFactory{
		IntegrationsStorage:   integrationsStorage,
		AuthorizationsStorage: authorizationsStorage,
		OAuthProvider:         oauthProvider,
	}
	pipeService := NewService(
		oauthProvider,
		pipesStorage,
		integrationsStorage,
		authorizationsStorage,
		idMappingsStore,
		importsStorage,
		pipesQueue,
		api,
		authFactory,
		cfg.PipesAPIHost,
	)

	integrations, err := pipeService.GetIntegrations(workspaceID)

	if err != nil {
		t.Fatalf("workspaceIntegrations returned error: %v", err)
	}

	want := [][]*domain.Pipe{
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

func (ts *ServiceTestSuite) TestService_Set_GetAvailableAuthorizations() {

	s := pipeStorage.NewPostgresStorage(ts.db)
	as := authorizationStorage.NewPostgresStorage(ts.db)
	ims := importStorage.NewPostgresStorage(ts.db)
	idms := idMappingStorage.NewPostgresStorage(ts.db)
	api := client.NewTogglApiClient("https://localhost")
	op := &mocks.OAuthProvider{}
	q := &mocks.Queue{}

	is := &mocks.IntegrationsStorage{}
	is.On("SaveAuthorizationType", mock.Anything, mock.Anything).Return(nil)
	is.On("LoadAuthorizationType", integration.GitHub).Return(domain.TypeOauth2, nil)
	is.On("LoadAuthorizationType", integration.Asana).Return(domain.TypeOauth1, nil)

	af := &domain.AuthorizationFactory{
		IntegrationsStorage:   is,
		AuthorizationsStorage: as,
		OAuthProvider:         op,
	}

	svc := NewService(op, s, is, as, idms, ims, q, api, af, "https://localhost")

	res, err := svc.integrationsStore.LoadAuthorizationType(integration.GitHub)
	ts.NoError(err)
	ts.Equal(domain.TypeOauth2, res)

	err = svc.integrationsStore.SaveAuthorizationType(integration.GitHub, domain.TypeOauth2)
	ts.NoError(err)
	err = svc.integrationsStore.SaveAuthorizationType(integration.Asana, domain.TypeOauth1)
	ts.NoError(err)

	res, err = svc.integrationsStore.LoadAuthorizationType(integration.GitHub)
	ts.NoError(err)
	ts.Equal(domain.TypeOauth2, res)

	res, err = svc.integrationsStore.LoadAuthorizationType(integration.Asana)
	ts.NoError(err)
	ts.Equal(domain.TypeOauth1, res)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ServiceTestSuite))
}
