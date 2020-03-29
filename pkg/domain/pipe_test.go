package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/toggl/pipes-api/pkg/domain"
)

func TestNewPipe(t *testing.T) {
	p := domain.NewPipe(1, domain.GitHub, domain.ProjectsPipe)
	assert.Equal(t, "github:projects", p.Key())
	assert.Equal(t, domain.GitHub, p.ServiceID)
	assert.Equal(t, 1, p.WorkspaceID)
}

func TestPipesKey(t *testing.T) {
	pk := domain.PipesKey(domain.GitHub, domain.ProjectsPipe)
	assert.Equal(t, "github:projects", pk)
}

func TestGetSidPidFromKey(t *testing.T) {
	sid, pid := domain.GetSidPidFromKey("github:projects")
	assert.Equal(t, domain.GitHub, sid)
	assert.Equal(t, domain.ProjectsPipe, pid)
}
