package integration

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tambet/oauthplain"

	"github.com/toggl/pipes-api/pkg/domain"
)

func TestFreshbooks_WorkspaceID(t *testing.T) {
	s := &FreshBooksPipeIntegration{WorkspaceID: 1}
	assert.Equal(t, 1, s.GetWorkspaceID())
}

func TestFreshbooks_ID(t *testing.T) {
	s := &FreshBooksPipeIntegration{}
	assert.Equal(t, domain.FreshBooks, s.ID())
}

func TestFreshbooks_KeyFor(t *testing.T) {
	tests := []struct {
		want string
		got  domain.PipeID
	}{
		{want: "freshbooks:users", got: domain.UsersPipe},
		{want: "freshbooks:clients", got: domain.ClientsPipe},
		{want: "freshbooks:projects", got: domain.ProjectsPipe},
		{want: "freshbooks:tasks", got: domain.TasksPipe},
		{want: "freshbooks:todolists", got: domain.TodoListsPipe},
		{want: "freshbooks:todos", got: domain.TodosPipe},
		{want: "freshbooks:timeentries", got: domain.TimeEntriesPipe},
		{want: "freshbooks:accounts", got: domain.AccountsPipe},
	}

	svc := &FreshBooksPipeIntegration{}
	for _, v := range tests {
		assert.Equal(t, v.want, svc.KeyFor(v.got))
	}
}

func TestFreshbooks_SetAuthData(t *testing.T) {
	s := &FreshBooksPipeIntegration{}
	token := oauthplain.Token{
		ConsumerKey:      "",
		ConsumerSecret:   "",
		OAuthToken:       "",
		OAuthTokenSecret: "",
		OAuthVerifier:    "",
		AuthorizeUrl:     "",
		Extra:            map[string]string{"account_name": "testAccount"},
	}
	b, err := json.Marshal(token)
	assert.NoError(t, err)

	err = s.SetAuthData(b)
	assert.NoError(t, err)
	assert.Equal(t, token, s.token)
	assert.Equal(t, "testAccount", s.accountName)

	b2, err := json.Marshal("wrong_data")
	assert.NoError(t, err)

	err = s.SetAuthData(b2)
	assert.Error(t, err)
}

func TestFreshbooks_SetParams(t *testing.T) {
	s := &FreshBooksPipeIntegration{}
	err := s.SetParams([]byte("any"))
	assert.NoError(t, err)
}

func TestFreshbooks_SetSince(t *testing.T) {
	s := &FreshBooksPipeIntegration{}
	s.SetSince(&time.Time{})
}

func TestIntegration_Freshbooks_Accounts(t *testing.T) {
	s := &FreshBooksPipeIntegration{}
	c, err := s.Accounts()
	assert.NoError(t, err)
	assert.Equal(t, []*domain.Account{}, c)
}

func TestIntegration_Freshbooks_TodoLists(t *testing.T) {
	s := &FreshBooksPipeIntegration{}
	te, err := s.TodoLists()
	assert.NoError(t, err)
	assert.Equal(t, []*domain.Task{}, te)
}

func TestFreshbooks_numberStrToInt64(t *testing.T) {
	v := numberStrToInt("10")
	assert.Equal(t, 10, v)

	v2 := numberStrToInt("unknown")
	assert.Equal(t, 0, v2)
}
