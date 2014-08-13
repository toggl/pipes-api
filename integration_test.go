package main

import (
  "reflect"
	"testing"
)

var testDB = "pipes_test"

func init() {
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
    {ID: "basecamp", Name: "Basecamp", Link: "http://support.toggl.com/basecamp", Image:"/images/logo-basecamp.png", AuthType:"oauth2"},
    {ID: "freshbooks", Name: "Freshbooks", Link: "http://support.toggl.com/freshbooks", Image:"/images/logo-freshbooks.png", AuthType:"oauth1"},
    {ID: "teamweek", Name: "Teamweek", Link: "http://support.toggl.com/teamweek", Image:"/images/logo-teamweek.png", AuthType:"oauth2"},
    {ID: "asana", Name: "Asana", Link: "http://support.toggl.com/asana", Image:"/images/logo-asana.png", AuthType:"oauth2"},
  }

  if !reflect.DeepEqual(want, integrations) {
    t.Errorf("workspaceIntegrations returned %+v, want %+v", integrations, want)
  }
}
