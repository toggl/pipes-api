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
