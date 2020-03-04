package integrations

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/toggl/pipes-api/pkg/integrations/asana"
	"github.com/toggl/pipes-api/pkg/integrations/basecamp"
	"github.com/toggl/pipes-api/pkg/integrations/freshbooks"
	"github.com/toggl/pipes-api/pkg/integrations/github"
	"github.com/toggl/pipes-api/pkg/integrations/teamweek"
)

func TestNewExternalService(t *testing.T) {
	s1 := NewExternalService(basecamp.ServiceID, 1)
	s2 := NewExternalService(asana.ServiceID, 2)
	s3 := NewExternalService(github.ServiceID, 3)
	s4 := NewExternalService(freshbooks.ServiceID, 4)
	s5 := NewExternalService(teamweek.ServiceID, 5)

	assert.Equal(t, basecamp.ServiceID, s1.ID())
	assert.Equal(t, asana.ServiceID, s2.ID())
	assert.Equal(t, github.ServiceID, s3.ID())
	assert.Equal(t, freshbooks.ServiceID, s4.ID())
	assert.Equal(t, teamweek.ServiceID, s5.ID())
}

func TestNewExternalServicePanic(t *testing.T) {
	pf := func() { NewExternalService("Unknown", 1) }
	assert.Panics(t, pf)
}
