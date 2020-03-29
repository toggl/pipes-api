package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/toggl/pipes-api/pkg/domain"
	"github.com/toggl/pipes-api/pkg/integration"
)

func TestNewPipe(t *testing.T) {
	p := domain.NewPipe(1, integration.GitHub, integration.ProjectsPipe)
	assert.Equal(t, "github:projects", p.Key())
	assert.Equal(t, integration.GitHub, p.ServiceID)
	assert.Equal(t, 1, p.WorkspaceID)
}

func TestPipesKey(t *testing.T) {
	pk := domain.PipesKey(integration.GitHub, integration.ProjectsPipe)
	assert.Equal(t, "github:projects", pk)
}

func TestGetSidPidFromKey(t *testing.T) {
	sid, pid := domain.GetSidPidFromKey("github:projects")
	assert.Equal(t, integration.GitHub, sid)
	assert.Equal(t, integration.ProjectsPipe, pid)
}
