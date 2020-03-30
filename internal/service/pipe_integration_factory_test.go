package service_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/toggl/pipes-api/internal/service"
	"github.com/toggl/pipes-api/pkg/domain"
)

func TestNewPipeIntegration(t *testing.T) {
	s1 := service.NewPipeIntegration(domain.BaseCamp, 1)
	s2 := service.NewPipeIntegration(domain.Asana, 2)
	s3 := service.NewPipeIntegration(domain.GitHub, 3)
	s4 := service.NewPipeIntegration(domain.FreshBooks, 4)
	s5 := service.NewPipeIntegration(domain.TeamWeek, 5)

	assert.Equal(t, domain.BaseCamp, s1.ID())
	assert.Equal(t, domain.Asana, s2.ID())
	assert.Equal(t, domain.GitHub, s3.ID())
	assert.Equal(t, domain.FreshBooks, s4.ID())
	assert.Equal(t, domain.TeamWeek, s5.ID())
}

func TestNewPipeIntegrationPanic(t *testing.T) {
	pf := func() { service.NewPipeIntegration("Unknown", 1) }
	assert.Panics(t, pf)
}
