package pipes

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/toggl/pipes-api/pkg/cfg"
	"github.com/toggl/pipes-api/pkg/integrations/mock"
	"github.com/toggl/pipes-api/pkg/storage"
	"github.com/toggl/pipes-api/pkg/toggl"
)

var (
	workspaceID = 1
	pipeID      = "users"
	serviceID   = "basecamp"
)

func TestNewClient(t *testing.T) {
	expectedKey := "basecamp:users"
	p := cfg.NewPipe(workspaceID, serviceID, pipeID)

	if p.Key != expectedKey {
		t.Errorf("NewPipe Key = %v, want %v", p.Key, expectedKey)
	}
}

func TestPipeEndSyncJSONParsingFail(t *testing.T) {
	flags := cfg.Flags{}
	cfg.ParseFlags(&flags)

	oauthService := &cfg.OAuthService{
		Environment: flags.Environment,
	}

	store := &storage.Storage{ConnString: flags.TestDBConnString}
	store.Connect()
	defer store.Close()

	authService := &AuthorizationService{
		Storage:                 store,
		AvailableAuthorizations: cfg.AvailableAuthorizations, // TODO: Remove global state
		Environment:             flags.Environment,
		OAuth2Configs:           cfg.OAuth2Configs, // TODO: Remove global state
	}

	togglService := &toggl.Service{
		URL: cfg.Urls.TogglAPIHost[flags.Environment], // TODO: Remove Global state
	}

	connService := &ConnectionService{
		Storage: store,
	}

	pipeService := &PipeService{
		Storage:               store,
		AuthorizationService:  authService,
		TogglService:          togglService,
		ConnectionService:     connService,
		PipesApiHost:          cfg.Urls.PipesAPIHost[flags.Environment], // TODO: Remove Global state
		AvailableIntegrations: cfg.AvailableIntegrations,                // TODO: Remove Global state
		OAuthService:          oauthService,
	}

	p := cfg.NewPipe(workspaceID, mock.ServiceName, projectsPipeID)

	jsonUnmarshalError := &json.UnmarshalTypeError{
		Value:  "asd",
		Type:   reflect.TypeOf(1),
		Struct: "Project",
		Field:  "id",
	}
	if err := pipeService.NewStatus(p); err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	if err := pipeService.endSync(p, true, jsonUnmarshalError); err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	if p.PipeStatus.Message != cfg.ErrJSONParsing.Error() {
		t.Fatalf(
			"FailPipe expected to get wrapper error %s, but get %s",
			cfg.ErrJSONParsing.Error(), p.PipeStatus.Message,
		)
	}
}

func TestGetPipesFromQueue_DoesNotReturnMultipleSameWorkspace(t *testing.T) {
	flags := cfg.Flags{}
	cfg.ParseFlags(&flags)

	store := &storage.Storage{ConnString: flags.TestDBConnString}
	store.Connect()
	defer store.Close()

	oauthService := &cfg.OAuthService{
		Environment: flags.Environment,
	}

	authService := &AuthorizationService{
		Storage:                 store,
		AvailableAuthorizations: cfg.AvailableAuthorizations, // TODO: Remove global state
		Environment:             flags.Environment,
		OAuth2Configs:           cfg.OAuth2Configs, // TODO: Remove global state
	}

	togglService := &toggl.Service{
		URL: cfg.Urls.TogglAPIHost[flags.Environment], // TODO: Remove Global state
	}

	connService := &ConnectionService{
		Storage: store,
	}

	pipeService := &PipeService{
		Storage:               store,
		AuthorizationService:  authService,
		TogglService:          togglService,
		ConnectionService:     connService,
		PipesApiHost:          cfg.Urls.PipesAPIHost[flags.Environment], // TODO: Remove Global state
		AvailableIntegrations: cfg.AvailableIntegrations,                // TODO: Remove Global state
		OAuthService:          oauthService,
	}

	createAndEnqueuePipeFn := func(workspaceID int, serviceID, pipeID string, priority int) *cfg.Pipe {
		pipe := cfg.NewPipe(workspaceID, serviceID, pipeID)
		pipe.Automatic = true
		pipe.Configured = true
		data, err := json.Marshal(pipe)
		if err != nil {
			t.Error(err)
			return nil
		}
		_, err = store.Exec(`
			with created as (
				insert into pipes(workspace_id, Key, data)
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

	// first fetch should return 3 pipes and unique per workspace
	pipes, err := pipeService.GetPipesFromQueue()
	if err != nil {
		t.Error(err)
	}
	if len(pipes) != 3 {
		t.Error("should return 3 pipes")
	}
	if pipes[0].WorkspaceID != 3 {
		t.Error("first returned pipe should be pipe for workspace 3 because it has highest priority")
	}

	// make sure returned pipes are unique per workspace
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
