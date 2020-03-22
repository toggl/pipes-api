package basecamp

import (
	"encoding/json"
	"testing"
	"time"

	"code.google.com/p/goauth2/oauth"
	"github.com/stretchr/testify/assert"

	"github.com/toggl/pipes-api/pkg/integration"
	"github.com/toggl/pipes-api/pkg/toggl"
)

func TestBasecamp_WorkspaceID(t *testing.T) {
	s := &Service{WorkspaceID: 1}
	assert.Equal(t, 1, s.GetWorkspaceID())
}

func TestBasecamp_ID(t *testing.T) {
	s := &Service{}
	assert.Equal(t, integration.BaseCamp, s.ID())
}

func TestBasecamp_KeyFor(t *testing.T) {
	s := &Service{}
	assert.Equal(t, "basecamp:account:accounts", s.KeyFor(integration.AccountsPipe))

	tests := []struct {
		want string
		got  integration.PipeID
	}{
		{want: "basecamp:account:1:users", got: integration.UsersPipe},
		{want: "basecamp:account:1:clients", got: integration.ClientsPipe},
		{want: "basecamp:account:1:projects", got: integration.ProjectsPipe},
		{want: "basecamp:account:1:tasks", got: integration.TasksPipe},
		{want: "basecamp:account:1:todolists", got: integration.TodoListsPipe},
		{want: "basecamp:account:1:todos", got: integration.TodosPipe},
		{want: "basecamp:account:1:timeentries", got: integration.TimeEntriesPipe},
		{want: "basecamp:account:1:accounts", got: integration.AccountsPipe},
	}

	svc := &Service{BasecampParams: &BasecampParams{AccountID: 1}}
	for _, v := range tests {
		assert.Equal(t, v.want, svc.KeyFor(v.got))
	}
}

func TestBasecamp_SetAuthData(t *testing.T) {
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

func TestBasecamp_SetParams(t *testing.T) {
	s := &Service{}
	ap := BasecampParams{AccountID: 5}
	b, err := json.Marshal(ap)
	assert.NoError(t, err)

	err = s.SetParams(b)
	assert.NoError(t, err)
	assert.Equal(t, ap, *s.BasecampParams)

	b2, err := json.Marshal(BasecampParams{AccountID: 0})
	assert.NoError(t, err)

	s2 := &Service{}
	err = s2.SetParams(b2)
	assert.Error(t, err)

	b3, err := json.Marshal("")
	assert.NoError(t, err)

	s3 := &Service{}
	err = s3.SetParams(b3)
	assert.Error(t, err)
}

func TestBasecamp_SetSince(t *testing.T) {
	s := &Service{}
	s.SetSince(&time.Time{})
}

func TestIntegration_Basecamp_Clients(t *testing.T) {
	s := &Service{}
	c, err := s.Clients()
	assert.NoError(t, err)
	assert.Equal(t, []*toggl.Client{}, c)
}

func TestIntegration_Basecamp_ExportTimeEntry(t *testing.T) {
	s := &Service{}
	te, err := s.ExportTimeEntry(&toggl.TimeEntry{})
	assert.NoError(t, err)
	assert.Equal(t, 0, te)
}
