package pipes

import (
	"database/sql"
	"encoding/json"
	"log"
	"reflect"
	"strings"
	"testing"

	"github.com/toggl/pipes-api/pkg/authorization"
	"github.com/toggl/pipes-api/pkg/connection"
	"github.com/toggl/pipes-api/pkg/environment"
	"github.com/toggl/pipes-api/pkg/integrations/mock"
	"github.com/toggl/pipes-api/pkg/toggl"
)

var (
	workspaceID = 1
	pipeID      = "users"
	serviceID   = "basecamp"
)

func TestNewClient(t *testing.T) {
	expectedKey := "basecamp:users"
	p := NewPipe(workspaceID, serviceID, pipeID)

	if p.Key != expectedKey {
		t.Errorf("NewPipe Key = %v, want %v", p.Key, expectedKey)
	}
}

func TestPipeEndSyncJSONParsingFail(t *testing.T) {
	flags := environment.Flags{}
	environment.ParseFlags(&flags)

	cfgService := environment.New(flags.Environment, flags.WorkDir)

	db, err := sql.Open("postgres", flags.TestDBConnString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	api := toggl.NewApiClient(cfgService.GetTogglAPIHost())

	authStore := authorization.NewStorage(db, cfgService)
	connStore := connection.NewStorage(db)

	pipesStorage := NewStorage(cfgService, db)
	pipeService := NewService(cfgService, authStore, pipesStorage, connStore, api)

	p := NewPipe(workspaceID, mock.ServiceName, projectsPipeID)

	jsonUnmarshalError := &json.UnmarshalTypeError{
		Value:  "asd",
		Type:   reflect.TypeOf(1),
		Struct: "Project",
		Field:  "id",
	}
	if err := pipeService.newStatus(p); err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	if err := pipeService.endSync(p, true, jsonUnmarshalError); err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	if p.PipeStatus.Message != ErrJSONParsing.Error() {
		t.Fatalf(
			"FailPipe expected to get wrapper error %s, but get %s",
			ErrJSONParsing.Error(), p.PipeStatus.Message,
		)
	}
}

func TestGetPipesFromQueue_DoesNotReturnMultipleSameWorkspace(t *testing.T) {
	flags := environment.Flags{}
	environment.ParseFlags(&flags)

	cfgService := environment.New(flags.Environment, flags.WorkDir)

	db, err := sql.Open("postgres", flags.TestDBConnString)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	api := toggl.NewApiClient(cfgService.GetTogglAPIHost())

	authStore := authorization.NewStorage(db, cfgService)
	connStore := connection.NewStorage(db)

	pipesStorage := NewStorage(cfgService, db)
	pipeService := NewService(cfgService, authStore, pipesStorage, connStore, api)

	createAndEnqueuePipeFn := func(workspaceID int, serviceID, pipeID string, priority int) *Pipe {
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

func TestGetProjects(t *testing.T) {
	flags := environment.Flags{}
	environment.ParseFlags(&flags)

	cfgService := environment.New(flags.Environment, flags.WorkDir)

	db, err := sql.Open("postgres", flags.TestDBConnString)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()

	api := toggl.NewApiClient(cfgService.GetTogglAPIHost())
	authStore := authorization.NewStorage(db, cfgService)
	connStore := connection.NewStorage(db)

	pipesStorage := NewStorage(cfgService, db)
	pipeService := NewService(cfgService, authStore, pipesStorage, connStore, api)

	p := NewPipe(1, mock.ServiceName, "projects")

	err = pipeService.fetchProjects(p)
	if err != nil {
		t.Error(err)
	}

	service := Create(p.ServiceID, p.WorkspaceID)
	s, err := pipeService.auth.IntegrationFor(service, p.ServiceParams)
	if err != nil {
		t.Error(err)
	}
	b, err := pipesStorage.getObject(s, "projects")
	if err != nil {
		t.Error(err)
	}
	var pr toggl.ProjectsResponse
	err = json.Unmarshal(b, &pr)
	if err != nil {
		t.Error(err)
	}
	if len(pr.Projects) != 4 {
		t.Errorf("Expected 4 projects but got %d", len(pr.Projects))
	}
	if pr.Projects[0].Name != strings.TrimSpace(mock.P1Name) {
		t.Errorf("Expected name '%s' but got '%s'", strings.TrimSpace(mock.P1Name), pr.Projects[0].Name)
	}
	if pr.Projects[1].Name != strings.TrimSpace(mock.P2Name) {
		t.Errorf("Expected name '%s' but got '%s'", strings.TrimSpace(mock.P2Name), pr.Projects[1].Name)
	}
	if pr.Projects[2].Name != strings.TrimSpace(mock.P3Name) {
		t.Errorf("Expected name '%s' but got '%s'", strings.TrimSpace(mock.P3Name), pr.Projects[2].Name)
	}
	if pr.Projects[3].Name != strings.TrimSpace(mock.P4Name) {
		t.Errorf("Expected name '%s' but got '%s'", strings.TrimSpace(mock.P4Name), pr.Projects[3].Name)
	}
}
