package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClientsImport_Count(t *testing.T) {
	ci := &ClientsImport{
		Clients: []*Client{
			{
				ID:        1,
				Name:      "1",
				ForeignID: "1",
			},
			{
				ID:        2,
				Name:      "2",
				ForeignID: "2",
			},
		},
		Notifications: []string{},
	}

	assert.Equal(t, 2, ci.Count())
}

func TestUsersImport_Count(t *testing.T) {
	ui := &UsersImport{
		WorkspaceUsers: []*User{
			{
				ID:             1,
				Email:          "1",
				Name:           "1",
				SendInvitation: false,
				ForeignID:      "1",
			},
			{
				ID:             2,
				Email:          "2",
				Name:           "2",
				SendInvitation: true,
				ForeignID:      "2",
			},
		},
		Notifications: []string{},
	}

	assert.Equal(t, 2, ui.Count())
}

func TestProjectsImport_Count(t *testing.T) {
	pi := &ProjectsImport{
		Projects: []*Project{
			{
				ID:              1,
				Name:            "1",
				Active:          false,
				Billable:        false,
				ClientID:        1,
				ForeignID:       "1",
				ForeignClientID: "1",
			},
			{
				ID:              2,
				Name:            "2",
				Active:          false,
				Billable:        false,
				ClientID:        2,
				ForeignID:       "2",
				ForeignClientID: "2",
			},
		},
		Notifications: []string{},
	}

	assert.Equal(t, 2, pi.Count())
}

func TestTasksImport_Count(t *testing.T) {
	ti := &TasksImport{
		Tasks: []*Task{
			{
				ID:               1,
				Name:             "1",
				Active:           false,
				ProjectID:        1,
				ForeignID:        "1",
				ForeignProjectID: "1",
			},
			{
				ID:               2,
				Name:             "2",
				Active:           false,
				ProjectID:        2,
				ForeignID:        "2",
				ForeignProjectID: "2",
			},
		},
		Notifications: []string{},
	}

	assert.Equal(t, 2, ti.Count())
}
