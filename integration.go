package main

import (
	"strings"
)

type (
	Integration struct {
		ID         string  `json:"id"`
		Name       string  `json:"name"`
		Link       string  `json:"link"`
		Image      string  `json:"image"`
		Pipes      []*Pipe `json:"pipes"`
		AuthURL    string  `json:"auth_url,omitempty"`
		AuthType   string  `json:"auth_type,omitempty"`
		Authorized bool    `json:"authorized"`
	}
)

// FIXME: Refactor all settings to conf file
var availableIntegration = map[string][]string{
	"basecamp":   {"users", "projects", "todolists", "todos"},
	"freshbooks": {"users", "projects", "tasks", "timeentries"},
	"teamweek":   {"users", "projects", "tasks"},
	"asana":      {"users", "projects", "tasks"},
}

var availableAuthorizations = map[string]string{
	"freshbooks": "oauth1",
	"basecamp":   "oauth2",
	"teamweek":   "oauth2",
	"asana":      "oauth2",
}

var availableImages = map[string]string{
	"basecamp":   "/images/logo-basecamp.png",
	"freshbooks": "/images/logo-freshbooks.png",
	"teamweek":   "/images/logo-teamweek.png",
	"asana":      "/images/logo-asana.png",
}

var availableLinks = map[string]string{
	"basecamp":   "http://support.toggl.com/basecamp",
	"freshbooks": "http://support.toggl.com/freshbooks",
	"teamweek":   "http://support.toggl.com/teamweek",
	"asana":      "http://support.toggl.com/asana",
}

var availableDescriptions = map[string]string{
	"basecamp:users":         "Basecamp users will be imported as Toggl users. Existing users are matched by e-mail.",
	"basecamp:projects":      "Basecamp projects will be imported as Toggl projects. Existing projects are matched by name.",
	"basecamp:todolists":     "Basecamp todolists will be imported as Toggl tasks. Existing tasks are matched by name.",
	"basecamp:todos":         "Basecamp todos will be imported as Toggl tasks. Existing tasks are matched by name.",
	"freshbooks:users":       "Freshbooks users will be imported as Toggl users. Existing users are matched by e-mail.",
	"freshbooks:projects":    "Freshbooks projects will be imported as Toggl projects. Existing projects are matched by name.",
	"freshbooks:tasks":       "Freshbooks tasks will be imported as Toggl tasks. Existing tasks are matched by name.",
	"freshbooks:timeentries": "Toggl time entries that are assigned to Freshbooks tasks will be exported your Freshbooks timesheet.",
	"teamweek:users":         "Teamweek users will be imported as Toggl users. Existing users are matched by e-mail.",
	"teamweek:projects":      "Teamweek projects will be imported as Toggl projects. Existing projects are matched by name.",
	"teamweek:tasks":         "Teamweek tasks will be imported as Toggl tasks. Existing tasks are matched by name.",
	"asana:users":            "Asana users will be imported as Toggl users. Existing users are matched by e-mail.",
	"asana:projects":         "Asana projects will be imported as Toggl projects. Existing projects are matched by name.",
	"asana:tasks":            "Asana tasks will be imported as Toggl tasks. Existing tasks are matched by name.",
}

var automaticOptions = map[string]bool{
	"basecamp:users":         false,
	"basecamp:projects":      true,
	"basecamp:todolists":     true,
	"basecamp:todos":         true,
	"freshbooks:users":       false,
	"freshbooks:projects":    true,
	"freshbooks:tasks":       true,
	"freshbooks:timeentries": true,
	"teamweek:users":         false,
	"teamweek:projects":      true,
	"teamweek:tasks":         true,
	"asana:users":            false,
	"asana:projects":         true,
	"asana:tasks":            true,
}

var premiumOptions = map[string]bool{
	"basecamp:users":         false,
	"basecamp:projects":      false,
	"basecamp:todolists":     true,
	"basecamp:todos":         true,
	"freshbooks:users":       false,
	"freshbooks:projects":    false,
	"freshbooks:tasks":       true,
	"freshbooks:timeentries": true,
	"teamweek:users":         false,
	"teamweek:projects":      false,
	"teamweek:tasks":         true,
	"asana:users":            false,
	"asana:projects":         false,
	"asana:tasks":            true,
}

var pipeNames = map[string]string{
	"users":       "Users",
	"tasks":       "Tasks",
	"todos":       "Todos",
	"clients":     "Clients",
	"projects":    "Projects",
	"todolists":   "Todo lists",
	"timeentries": "Time entries export",
}

func NewIntegration(serviceName string) Integration {
	integration := Integration{
		ID:       serviceName,
		Name:     strings.Title(serviceName),
		Link:     availableLinks[serviceName],
		Image:    availableImages[serviceName],
		AuthType: availableAuthorizations[serviceName],
		AuthURL:  oAuth2URL(serviceName),
	}
	return integration
}

func workspaceIntegrations(workspaceID int) ([]Integration, error) {
	// FIXME: if authorizations, workspace pipes, pipe statues
	// don't block each others loading, load all 3 at the same time.

	authorizations, err := loadAuthorizations(workspaceID)
	if err != nil {
		return nil, err
	}

	workspacePipes, err := loadPipes(workspaceID)
	if err != nil {
		return nil, err
	}

	pipeStatuses, err := loadPipeStatuses(workspaceID)
	if err != nil {
		return nil, err
	}

	var integrations []Integration
	for serviceID, pipeIDs := range availableIntegration {
		integration := NewIntegration(serviceID)
		integration.Authorized = authorizations[serviceID]
		for _, pipeID := range pipeIDs {
			key := pipesKey(serviceID, pipeID)
			pipe := workspacePipes[key]
			if pipe == nil {
				pipe = NewPipe(workspaceID, serviceID, pipeID)
			}
			pipe.Name = pipeNames[pipeID]
			pipe.PipeStatus = pipeStatuses[key]
			pipe.Premium = premiumOptions[key]
			pipe.Description = availableDescriptions[key]
			pipe.AutomaticOption = automaticOptions[key]
			integration.Pipes = append(integration.Pipes, pipe)
		}
		integrations = append(integrations, integration)
	}
	return integrations, nil
}
