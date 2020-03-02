package storage

import (
	"database/sql"
	"encoding/json"
	"log"
	"reflect"
	"testing"

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
	p := environment.NewPipe(workspaceID, serviceID, pipeID)

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

	pipeService := NewPipesStorage(cfgService, api, db)

	p := environment.NewPipe(workspaceID, mock.ServiceName, projectsPipeID)

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

	if p.PipeStatus.Message != environment.ErrJSONParsing.Error() {
		t.Fatalf(
			"FailPipe expected to get wrapper error %s, but get %s",
			environment.ErrJSONParsing.Error(), p.PipeStatus.Message,
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

	pipeService := NewPipesStorage(cfgService, api, db)

	createAndEnqueuePipeFn := func(workspaceID int, serviceID, pipeID string, priority int) *environment.PipeConfig {
		pipe := environment.NewPipe(workspaceID, serviceID, pipeID)
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
