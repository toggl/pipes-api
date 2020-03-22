package github

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"code.google.com/p/goauth2/oauth"
	"github.com/stretchr/testify/assert"

	"github.com/toggl/pipes-api/pkg/integration"
	"github.com/toggl/pipes-api/pkg/toggl"
)

func TestGitHub_WorkspaceID(t *testing.T) {
	s := &Service{WorkspaceID: 1}
	assert.Equal(t, 1, s.GetWorkspaceID())
}

func TestGitHub_ID(t *testing.T) {
	s := &Service{}
	assert.Equal(t, integration.GitHub, s.ID())
}

func TestGitHub_KeyFor(t *testing.T) {
	tests := []struct {
		want string
		got  integration.PipeID
	}{
		{want: "github:users", got: integration.UsersPipe},
		{want: "github:clients", got: integration.ClientsPipe},
		{want: "github:projects", got: integration.ProjectsPipe},
		{want: "github:tasks", got: integration.TasksPipe},
		{want: "github:todolists", got: integration.TodoListsPipe},
		{want: "github:todos", got: integration.TodosPipe},
		{want: "github:timeentries", got: integration.TimeEntriesPipe},
		{want: "github:accounts", got: integration.AccountsPipe},
	}

	svc := &Service{}
	for _, v := range tests {
		assert.Equal(t, v.want, svc.KeyFor(v.got))
	}
}

func TestGitHub_SetAuthData(t *testing.T) {
	s := &Service{}
	token := oauth.Token{
		AccessToken:  "test",
		RefreshToken: "test2",
		Expiry:       time.Time{},
		Extra:        nil,
	}
	b, err := json.Marshal(token)
	assert.NoError(t, err)

	err = s.SetAuthData(b)
	assert.NoError(t, err)
	assert.Equal(t, token, s.token)

	b2, err := json.Marshal("wrong_data")
	assert.NoError(t, err)

	err = s.SetAuthData(b2)
	assert.Error(t, err)
}

func TestGitHub_SetParams(t *testing.T) {
	s := &Service{}
	b, err := json.Marshal([]byte("any"))
	assert.NoError(t, err)
	err = s.SetParams(b)
	assert.NoError(t, err)
}

func TestGitHub_SetSince(t *testing.T) {
	s := &Service{}
	s.SetSince(&time.Time{})
}

func TestIntegration_GitHub_Accounts(t *testing.T) {
	s := &Service{}
	c, err := s.Accounts()
	assert.NoError(t, err)
	assert.Equal(t, []*toggl.Account{{1, "Self"}}, c)
}

func TestIntegration_GitHub_Users(t *testing.T) {
	s := &Service{}
	c, err := s.Users()
	assert.NoError(t, err)
	assert.Equal(t, []*toggl.User{}, c)
}

func TestIntegration_GitHub_Tasks(t *testing.T) {
	s := &Service{}
	c, err := s.Tasks()
	assert.NoError(t, err)
	assert.Equal(t, []*toggl.Task{}, c)
}

func TestIntegration_GitHub_TodoLists(t *testing.T) {
	s := &Service{}
	c, err := s.TodoLists()
	assert.NoError(t, err)
	assert.Equal(t, []*toggl.Task{}, c)
}

func TestIntegration_GitHub_Clients(t *testing.T) {
	s := &Service{}
	c, err := s.Clients()
	assert.NoError(t, err)
	assert.Equal(t, []*toggl.Client{}, c)
}

func TestIntegration_GitHub_ExportTimeEntry(t *testing.T) {
	s := &Service{}
	te, err := s.ExportTimeEntry(&toggl.TimeEntry{})
	assert.NoError(t, err)
	assert.Equal(t, 0, te)
}

func TestIntegration_Github_Projects(t *testing.T) {
	testToken := os.Getenv("GITHUB_PERSONAL_TOKEN")
	if testToken == "" {
		t.Skipf("Skipped, because test environment variable GITHUB_PERSONAL_TOKEN hasn't been set")
	}

	s := &Service{}
	s.token = oauth.Token{AccessToken: testToken}

	projects, err := s.Projects()
	if err != nil {
		t.Error("error calling Projects, err:", err)
	}

	if len(projects) == 0 {
		t.Error("should return some projects")
	}
}
