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
