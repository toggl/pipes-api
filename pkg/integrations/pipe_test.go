package integrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPipe(t *testing.T) {
	p := NewPipe(1, "github", "projects")
	assert.Equal(t, "github:projects", p.Key)
	assert.Equal(t, "github", p.ServiceID)
	assert.Equal(t, 1, p.WorkspaceID)
}

func TestPipe_ValidatePayload(t *testing.T) {
	p := NewPipe(1, "github", "projects")
	out := p.ValidatePayload([]byte("test"))

	assert.Equal(t, "", out)

	p2 := NewPipe(1, "github", "users")
	out2 := p2.ValidatePayload([]byte("test"))

	assert.Equal(t, "", out2)
}

func TestPipe_ValidatePayload_MissingPayload(t *testing.T) {
	p := NewPipe(1, "github", "users")
	out := p.ValidatePayload([]byte(""))

	assert.Equal(t, "Missing request payload", out)
}

func TestPipesKey(t *testing.T) {
	pk := PipesKey("github", "projects")
	assert.Equal(t, "github:projects", pk)
}
