package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/toggl/pipes-api/pkg/domain"
	"github.com/toggl/pipes-api/pkg/integration"
)

func TestNewPipeStatus(t *testing.T) {
	s := domain.NewPipeStatus(1, "github", "projects", "https://store.toggl.space")

	assert.Equal(t, 1, s.WorkspaceID)
	assert.Equal(t, integration.GitHub, s.ServiceID)
	assert.Equal(t, integration.ProjectsPipe, s.PipeID)
	assert.Equal(t, domain.StatusRunning, s.Status)
	assert.Equal(t, time.Now().Format(time.RFC3339), s.SyncDate)
	assert.Equal(t, "github:projects", s.Key)
	assert.Equal(t, "https://store.toggl.space", s.PipesApiHost)
}

func TestStatus_AddError(t *testing.T) {
	s := domain.NewPipeStatus(1, "github", "projects", "https://store.toggl.space")
	s.AddError(errors.New("test error"))

	assert.Equal(t, domain.StatusError, s.Status)
	assert.Equal(t, "test error", s.Message)
}

func TestStatus_Complete(t *testing.T) {
	s := domain.NewPipeStatus(1, "github", "projects", "https://store.toggl.space")
	s.Complete("clients", []string{"test", "test2"}, 5)

	assert.Equal(t, domain.StatusSuccess, s.Status)
	assert.Equal(t, 2, len(s.Notifications))
	assert.Equal(t, 1, len(s.ObjectCounts))
	assert.Equal(t, "https://store.toggl.space/api/v1/integrations/github/pipes/projects/log", s.SyncLog)
}

func TestStatus_Complete_Err(t *testing.T) {
	s := domain.NewPipeStatus(1, "github", "projects", "https://store.toggl.space")
	s.AddError(errors.New("test error"))
	s.Complete("clients", []string{"test", "test2"}, 5)

	assert.Equal(t, domain.StatusError, s.Status)
	assert.Equal(t, 0, len(s.Notifications))
	assert.Equal(t, 0, len(s.ObjectCounts))
	assert.Equal(t, "", s.SyncLog)
}

func TestStatus_GenerateLog(t *testing.T) {
	s := domain.NewPipeStatus(1, "github", "projects", "https://store.toggl.space")
	s.Message = "msg"
	s.Notifications = []string{"notify1", "notify2"}
	log := s.GenerateLog()

	assert.Contains(t, log, "github")
	assert.Contains(t, log, "projects")
	assert.Contains(t, log, "msg")
	assert.Contains(t, log, "notify1")
	assert.Contains(t, log, "notify2")
}
