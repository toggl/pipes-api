package integration_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/toggl/pipes-api/pkg/domain"
	"github.com/toggl/pipes-api/pkg/integration"
)

func TestNewPipeIntegration(t *testing.T) {
	s1 := integration.NewPipeIntegration(domain.BaseCamp, 1)
	s2 := integration.NewPipeIntegration(domain.Asana, 2)
	s3 := integration.NewPipeIntegration(domain.GitHub, 3)
	s4 := integration.NewPipeIntegration(domain.FreshBooks, 4)
	s5 := integration.NewPipeIntegration(domain.TogglPlan, 5)

	assert.Equal(t, domain.BaseCamp, s1.ID())
	assert.Equal(t, domain.Asana, s2.ID())
	assert.Equal(t, domain.GitHub, s3.ID())
	assert.Equal(t, domain.FreshBooks, s4.ID())
	assert.Equal(t, domain.TogglPlan, s5.ID())
}

func TestNewPipeIntegrationPanic(t *testing.T) {
	pf := func() { integration.NewPipeIntegration("Unknown", 1) }
	assert.Panics(t, pf)
}
