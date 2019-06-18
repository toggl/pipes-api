package main

import (
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

func TestFailPipeResult(t *testing.T) {
	p := NewPipe(workspaceID, TestFailServiceName, projectsPipeID)

	if err := p.save(); err != nil {
		t.Errorf("Unexpected error %v", err)
		t.FailNow()
	}

	p.run()

	if p.PipeStatus.Message != ErrJSONParsing.Error() {
		t.Errorf("FailPipe expected to get wrapper error %s, but get %s", ErrJSONParsing.Error(), p.PipeStatus.Message)
	}
}
