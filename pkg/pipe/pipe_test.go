package pipe

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/toggl/pipes-api/pkg/integrations"
)

func TestNewPipe(t *testing.T) {
	p := NewPipe(1, integrations.GitHub, integrations.ProjectsPipe)
	assert.Equal(t, "github:projects", p.Key)
	assert.Equal(t, integrations.GitHub, p.ServiceID)
	assert.Equal(t, 1, p.WorkspaceID)
}

func TestPipesKey(t *testing.T) {
	pk := PipesKey(integrations.GitHub, integrations.ProjectsPipe)
	assert.Equal(t, "github:projects", pk)
}

func TestGetSidPidFromKey(t *testing.T) {
	sid, pid := GetSidPidFromKey("github:projects")
	assert.Equal(t, integrations.GitHub, sid)
	assert.Equal(t, integrations.ProjectsPipe, pid)
}

func TestNewExternalService(t *testing.T) {
	s1 := NewExternalService(integrations.BaseCamp, 1)
	s2 := NewExternalService(integrations.Asana, 2)
	s3 := NewExternalService(integrations.GitHub, 3)
	s4 := NewExternalService(integrations.FreshBooks, 4)
	s5 := NewExternalService(integrations.TeamWeek, 5)

	assert.Equal(t, integrations.BaseCamp, s1.ID())
	assert.Equal(t, integrations.Asana, s2.ID())
	assert.Equal(t, integrations.GitHub, s3.ID())
	assert.Equal(t, integrations.FreshBooks, s4.ID())
	assert.Equal(t, integrations.TeamWeek, s5.ID())
}

func TestNewExternalServicePanic(t *testing.T) {
	pf := func() { NewExternalService("Unknown", 1) }
	assert.Panics(t, pf)
}
