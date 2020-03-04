package integrations

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewPipeStatus(t *testing.T) {
	s := NewPipeStatus(1, "github", "projects", "https://pipes.toggl.space")

	assert.Equal(t, 1, s.WorkspaceID)
	assert.Equal(t, "github", s.ServiceID)
	assert.Equal(t, "projects", s.PipeID)
	assert.Equal(t, StatusRunning, s.Status)
	assert.Equal(t, time.Now().Format(time.RFC3339), s.SyncDate)
	assert.Equal(t, "github:projects", s.Key)
	assert.Equal(t, "https://pipes.toggl.space", s.PipesApiHost)
}

func TestStatus_AddError(t *testing.T) {
	s := NewPipeStatus(1, "github", "projects", "https://pipes.toggl.space")
	s.AddError(errors.New("test error"))

	assert.Equal(t, StatusError, s.Status)
	assert.Equal(t, "test error", s.Message)
}

func TestStatus_Complete(t *testing.T) {
	s := NewPipeStatus(1, "github", "projects", "https://pipes.toggl.space")
	s.Complete("clients", []string{"test", "test2"}, 5)

	assert.Equal(t, StatusSuccess, s.Status)
	assert.Equal(t, 2, len(s.Notifications))
	assert.Equal(t, 1, len(s.ObjectCounts))
	assert.Equal(t, "https://pipes.toggl.space/api/v1/integrations/github/pipes/projects/log", s.SyncLog)
}

func TestStatus_Complete_Err(t *testing.T) {
	s := NewPipeStatus(1, "github", "projects", "https://pipes.toggl.space")
	s.AddError(errors.New("test error"))
	s.Complete("clients", []string{"test", "test2"}, 5)

	assert.Equal(t, StatusError, s.Status)
	assert.Equal(t, 0, len(s.Notifications))
	assert.Equal(t, 0, len(s.ObjectCounts))
	assert.Equal(t, "", s.SyncLog)
}

func TestStatus_GenerateLog(t *testing.T) {
	s := NewPipeStatus(1, "github", "projects", "https://pipes.toggl.space")
	s.Message = "msg"
	s.Notifications = []string{"notify1", "notify2"}
	log := s.GenerateLog()

	assert.Contains(t, log, "github")
	assert.Contains(t, log, "projects")
	assert.Contains(t, log, "msg")
	assert.Contains(t, log, "notify1")
	assert.Contains(t, log, "notify2")
}
