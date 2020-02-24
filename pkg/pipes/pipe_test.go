package pipes

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/toggl/pipes-api/pkg/cfg"
	"github.com/toggl/pipes-api/pkg/integrations/mock"
	"github.com/toggl/pipes-api/pkg/storage"
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
	p := NewPipe(workspaceID, mock.ServiceName, projectsPipeID)

	jsonUnmarshalError := &json.UnmarshalTypeError{
		Value:  "asd",
		Type:   reflect.TypeOf(1),
		Struct: "Project",
		Field:  "id",
	}
	if err := p.NewStatus(); err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	if err := p.endSync(true, jsonUnmarshalError); err != nil {
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
	config := cfg.Flags{}
	cfg.ParseFlags(&config)

	db := storage.Storage{ConnString: config.TestDBConnString}
	db.Connect()
	defer db.Close()

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
		`, pipe.workspaceID, pipe.Key, data, priority)
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
	pipes, err := getPipesFromQueue()
	if err != nil {
		t.Error(err)
	}
	if len(pipes) != 3 {
		t.Error("should return 3 pipes")
	}
	if pipes[0].workspaceID != 3 {
		t.Error("first returned pipe should be pipe for workspace 3 because it has highest priority")
	}

	// make sure returned pipes are unique per workspace
	retrievedWorkspace := map[int]bool{}
	for _, pipe := range pipes {
		_, exists := retrievedWorkspace[pipe.workspaceID]
		if exists {
			t.Error("there's already existing queued pipe with workspace id ", pipe.workspaceID)
		}

		retrievedWorkspace[pipe.workspaceID] = true
		err = setQueuedPipeSynced(pipe)
		if err != nil {
			t.Error(err)
		}
	}

	// second fetch should return 1 pipe left
	pipes, err = getPipesFromQueue()
	if err != nil {
		t.Error(err)
	}
	if len(pipes) != 1 {
		t.Error("should only return 1 pipe")
	}
	if pipes[0].workspaceID != 1 {
		t.Error("should be workspace 1")
	}
	if _, exists := retrievedWorkspace[pipes[0].workspaceID]; !exists {
		t.Error("should return pipe with workspace from retrievedWorkspace")
	}
}
