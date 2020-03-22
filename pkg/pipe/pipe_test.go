package pipe

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/toggl/pipes-api/pkg/integration"
)

func TestNewPipe(t *testing.T) {
	p := NewPipe(1, integration.GitHub, integration.ProjectsPipe)
	assert.Equal(t, "github:projects", p.Key)
	assert.Equal(t, integration.GitHub, p.ServiceID)
	assert.Equal(t, 1, p.WorkspaceID)
}

func TestPipesKey(t *testing.T) {
	pk := PipesKey(integration.GitHub, integration.ProjectsPipe)
	assert.Equal(t, "github:projects", pk)
}

func TestGetSidPidFromKey(t *testing.T) {
	sid, pid := GetSidPidFromKey("github:projects")
	assert.Equal(t, integration.GitHub, sid)
	assert.Equal(t, integration.ProjectsPipe, pid)
}

func TestNewExternalService(t *testing.T) {
	s1 := NewExternalService(integration.BaseCamp, 1)
	s2 := NewExternalService(integration.Asana, 2)
	s3 := NewExternalService(integration.GitHub, 3)
	s4 := NewExternalService(integration.FreshBooks, 4)
	s5 := NewExternalService(integration.TeamWeek, 5)

	assert.Equal(t, integration.BaseCamp, s1.ID())
	assert.Equal(t, integration.Asana, s2.ID())
	assert.Equal(t, integration.GitHub, s3.ID())
	assert.Equal(t, integration.FreshBooks, s4.ID())
	assert.Equal(t, integration.TeamWeek, s5.ID())
}

func TestNewExternalServicePanic(t *testing.T) {
	pf := func() { NewExternalService("Unknown", 1) }
	assert.Panics(t, pf)
}
