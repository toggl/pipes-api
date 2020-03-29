package teamweek

import (
	"encoding/json"
	"testing"
	"time"

	"code.google.com/p/goauth2/oauth"
	"github.com/stretchr/testify/assert"

	"github.com/toggl/pipes-api/pkg/domain"
)

func TestTeamWeek_WorkspaceID(t *testing.T) {
	s := &Service{WorkspaceID: 1}
	assert.Equal(t, 1, s.GetWorkspaceID())
}

func TestTeamWeek_ID(t *testing.T) {
	s := &Service{}
	assert.Equal(t, domain.TeamWeek, s.ID())
}

func TestTeamWeek_KeyFor(t *testing.T) {
	s := &Service{}
	assert.Equal(t, "teamweek:account:accounts", s.KeyFor(domain.AccountsPipe))

	tests := []struct {
		want string
		got  domain.PipeID
	}{
		{want: "teamweek:account:1:users", got: domain.UsersPipe},
		{want: "teamweek:account:1:clients", got: domain.ClientsPipe},
		{want: "teamweek:account:1:projects", got: domain.ProjectsPipe},
		{want: "teamweek:account:1:tasks", got: domain.TasksPipe},
		{want: "teamweek:account:1:todolists", got: domain.TodoListsPipe},
		{want: "teamweek:account:1:todos", got: domain.TodosPipe},
		{want: "teamweek:account:1:timeentries", got: domain.TimeEntriesPipe},
		{want: "teamweek:account:1:accounts", got: domain.AccountsPipe},
	}

	svc := &Service{TeamweekParams: &TeamweekParams{AccountID: 1}}
	for _, v := range tests {
		assert.Equal(t, v.want, svc.KeyFor(v.got))
	}
}

func TestTeamWeek_SetAuthData(t *testing.T) {
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

func TestTeamWeek_SetParams(t *testing.T) {
	s := &Service{}
	ap := TeamweekParams{AccountID: 5}
	b, err := json.Marshal(ap)
	assert.NoError(t, err)

	err = s.SetParams(b)
	assert.NoError(t, err)
	assert.Equal(t, ap, *s.TeamweekParams)

	b2, err := json.Marshal(TeamweekParams{AccountID: 0})
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

func TestTeamWeek_SetSince(t *testing.T) {
	s := &Service{}
	s.SetSince(&time.Time{})
}

func TestIntegration_TeamWeek_TodoLists(t *testing.T) {
	s := &Service{}
	c, err := s.TodoLists()
	assert.NoError(t, err)
	assert.Equal(t, []*domain.Task{}, c)
}

func TestIntegration_TeamWeek_Clients(t *testing.T) {
	s := &Service{}
	c, err := s.Clients()
	assert.NoError(t, err)
	assert.Equal(t, []*domain.Client{}, c)
}

func TestIntegration_TeamWeek_ExportTimeEntry(t *testing.T) {
	s := &Service{}
	te, err := s.ExportTimeEntry(&domain.TimeEntry{})
	assert.NoError(t, err)
	assert.Equal(t, 0, te)
}
