package main

import (
	"reflect"
	"testing"
)

var testDB = "pipes_test"

func init() {
	loadIntegrations()
	db = connectDB(*dbHost, *dbPort, testDB, *dbUser, *dbPass)
}

func TestWorkspaceIntegrations(t *testing.T) {
	integrations, err := workspaceIntegrations(workspaceID)

	if err != nil {
		t.Fatalf("workspaceIntegrations returned error: %v", err)
	}

	for i, _ := range integrations {
		integrations[i].Pipes = nil
	}

	want := []Integration{
		{ID: "basecamp", Name: "Basecamp", Link: "http://support.toggl.com/basecamp", Image: "/images/logo-basecamp.png", AuthType: "oauth2"},
		{ID: "freshbooks", Name: "Freshbooks", Link: "http://support.toggl.com/freshbooks", Image: "/images/logo-freshbooks.png", AuthType: "oauth1"},
		{ID: "teamweek", Name: "Teamweek", Link: "http://support.toggl.com/teamweek", Image: "/images/logo-teamweek.png", AuthType: "oauth2"},
		{ID: "asana", Name: "Asana", Link: "http://support.toggl.com/asana", Image: "/images/logo-asana.png", AuthType: "oauth2"},
	}

	if !reflect.DeepEqual(want, integrations) {
		t.Errorf("workspaceIntegrations returned %+v, want %+v", integrations, want)
	}
}

func TestWorkspaceIntegrationPipes(t *testing.T) {
	integrations, err := workspaceIntegrations(workspaceID)

	if err != nil {
		t.Fatalf("workspaceIntegrations returned error: %v", err)
	}

	want := [][]*Pipe{
		{ // Basecamp
			{ID: "users", Name: "Users", Premium: false, AutomaticOption: false, workspaceID: 1, serviceID: "basecamp", key: "basecamp:users"},
			{ID: "projects", Name: "Projects", Premium: false, AutomaticOption: true, workspaceID: 1, serviceID: "basecamp", key: "basecamp:projects"},
			{ID: "todolists", Name: "Todo lists", Premium: true, AutomaticOption: true, workspaceID: 1, serviceID: "basecamp", key: "basecamp:todolists"},
			{ID: "todos", Name: "Todos", Premium: true, AutomaticOption: true, workspaceID: 1, serviceID: "basecamp", key: "basecamp:todos"},
		},
		{ // Freshbooks
			{ID: "users", Name: "Users", Premium: false, AutomaticOption: false, workspaceID: 1, serviceID: "freshbooks", key: "freshbooks:users"},
			{ID: "projects", Name: "Projects", Premium: false, AutomaticOption: true, workspaceID: 1, serviceID: "freshbooks", key: "freshbooks:projects"},
			{ID: "tasks", Name: "Tasks", Premium: true, AutomaticOption: true, workspaceID: 1, serviceID: "freshbooks", key: "freshbooks:tasks"},
			{ID: "timeentries", Name: "Time entries", Premium: true, AutomaticOption: true, workspaceID: 1, serviceID: "freshbooks", key: "freshbooks:timeentries"},
		},
		{ // Teamweek
			{ID: "users", Name: "Users", Premium: false, AutomaticOption: false, workspaceID: 1, serviceID: "teamweek", key: "teamweek:users"},
			{ID: "projects", Name: "Projects", Premium: false, AutomaticOption: true, workspaceID: 1, serviceID: "teamweek", key: "teamweek:projects"},
			{ID: "tasks", Name: "Tasks", Premium: true, AutomaticOption: true, workspaceID: 1, serviceID: "teamweek", key: "teamweek:tasks"},
		},
		{ // Asana
			{ID: "users", Name: "Users", Premium: false, AutomaticOption: false, workspaceID: 1, serviceID: "asana", key: "asana:users"},
			{ID: "projects", Name: "Projects", Premium: false, AutomaticOption: true, workspaceID: 1, serviceID: "asana", key: "asana:projects"},
			{ID: "tasks", Name: "Tasks", Premium: true, AutomaticOption: true, workspaceID: 1, serviceID: "asana", key: "asana:tasks"},
		},
	}

	for i, _ := range integrations {
		for j, pipe := range integrations[i].Pipes {
			pipe.Description = ""
			if !reflect.DeepEqual(pipe, want[i][j]) {
				t.Errorf("workspaceIntegrations returned %+v, want %+v", pipe, want[i][j])
			}
		}
	}
}
