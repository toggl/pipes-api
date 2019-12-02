package main

import (
	"encoding/json"
	"reflect"
	"testing"
)

var (
	workspaceID = 1
	pipeID      = "users"
	serviceID   = "basecamp"
)

func TestNewClient(t *testing.T) {
	expectedKey := "basecamp:users"
	p := NewPipe(workspaceID, serviceID, pipeID)

	if p.key != expectedKey {
		t.Errorf("NewPipe key = %v, want %v", p.key, expectedKey)
	}
}

func TestPipeEndSyncJSONParsingFail(t *testing.T) {
	p := NewPipe(workspaceID, TestServiceName, projectsPipeID)

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
	db = connectDB(testDBConnString)
	createPipeFn := func(workspaceID int, serviceID, pipeID string) *Pipe {
		pipe := NewPipe(workspaceID, serviceID, pipeID)
		pipe.Automatic = true
		pipe.Configured = true
		data, err := json.Marshal(pipe)
		if err != nil {
			t.Error(err)
			return nil
		}
		_, err = db.Exec(`
			insert into pipes(workspace_id, key, data)
			values($1, $2, $3)
		`, pipe.workspaceID, pipe.key, data)
		if err != nil {
			t.Error(err)
		}
		return pipe
	}

	createPipeFn(1, "asana", "users")
	createPipeFn(2, "asana", "projects")
	createPipeFn(1, "asana", "projects")

	// schedule them to be processed automatically
	_, err := db.Exec(queueAutomaticPipesSQL)
	if err != nil {
		t.Error(err)
	}

	// first fetch should return 2 pipes (non duplicate workspace)
	pipes, err := getPipesFromQueue()
	if err != nil {
		t.Error(err)
	}
	if len(pipes) != 2 {
		t.Error("should only return 2 pipes")
	}

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
	if _, exists := retrievedWorkspace[pipes[0].workspaceID]; !exists {
		t.Error("should return pipe with workspace from retrievedWorkspace")
	}
}
