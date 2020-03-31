package integration

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"code.google.com/p/goauth2/oauth"
	"github.com/stretchr/testify/assert"

	"github.com/toggl/pipes-api/pkg/domain"
)

func TestAsana_WorkspaceID(t *testing.T) {
	s := &AsanaPipeIntegration{WorkspaceID: 1}
	assert.Equal(t, 1, s.GetWorkspaceID())
}

func TestAsana_ID(t *testing.T) {
	s := &AsanaPipeIntegration{}
	assert.Equal(t, domain.Asana, s.ID())
}

func TestAsana_KeyFor(t *testing.T) {
	s := &AsanaPipeIntegration{}
	assert.Equal(t, "asana:account:accounts", s.KeyFor(domain.AccountsPipe))

	tests := []struct {
		want string
		got  domain.PipeID
	}{
		{want: "asana:account:1:users", got: domain.UsersPipe},
		{want: "asana:account:1:clients", got: domain.ClientsPipe},
		{want: "asana:account:1:projects", got: domain.ProjectsPipe},
		{want: "asana:account:1:tasks", got: domain.TasksPipe},
		{want: "asana:account:1:todolists", got: domain.TodoListsPipe},
		{want: "asana:account:1:todos", got: domain.TodosPipe},
		{want: "asana:account:1:timeentries", got: domain.TimeEntriesPipe},
		{want: "asana:account:1:accounts", got: domain.AccountsPipe},
	}

	svc := &AsanaPipeIntegration{AsanaParams: &AsanaParams{AccountID: 1}}
	for _, v := range tests {
		assert.Equal(t, v.want, svc.KeyFor(v.got))
	}
}

func TestAsana_SetAuthData(t *testing.T) {
	s := &AsanaPipeIntegration{}
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

func TestAsana_SetParams(t *testing.T) {
	s := &AsanaPipeIntegration{}
	ap := AsanaParams{AccountID: 5}
	b, err := json.Marshal(ap)
	assert.NoError(t, err)

	err = s.SetParams(b)
	assert.NoError(t, err)
	assert.Equal(t, ap, *s.AsanaParams)

	b2, err := json.Marshal(AsanaParams{AccountID: 0})
	assert.NoError(t, err)

	s2 := &AsanaPipeIntegration{}
	err = s2.SetParams(b2)
	assert.Error(t, err)

	b3, err := json.Marshal("")
	assert.NoError(t, err)

	s3 := &AsanaPipeIntegration{}
	err = s3.SetParams(b3)
	assert.Error(t, err)
}

func TestAsana_numberStrToInt64(t *testing.T) {
	v := numberStrToInt64("10")
	assert.Equal(t, int64(10), v)

	v2 := numberStrToInt64("unknown")
	assert.Equal(t, int64(0), v2)
}

func TestAsana_SetSince(t *testing.T) {
	s := &AsanaPipeIntegration{}
	s.SetSince(&time.Time{})
}

func TestIntegration_Asana_Clients(t *testing.T) {
	s := &AsanaPipeIntegration{}
	c, err := s.Clients()
	assert.NoError(t, err)
	assert.Equal(t, []*domain.Client{}, c)
}

func TestIntegration_Asana_TodoLists(t *testing.T) {
	s := &AsanaPipeIntegration{}
	tl, err := s.TodoLists()
	assert.NoError(t, err)
	assert.Equal(t, []*domain.Task{}, tl)
}

func TestIntegration_Asana_ExportTimeEntry(t *testing.T) {
	s := &AsanaPipeIntegration{}
	te, err := s.ExportTimeEntry(&domain.TimeEntry{})
	assert.NoError(t, err)
	assert.Equal(t, 0, te)
}

func TestIntegration_Asana_Accounts(t *testing.T) {
	s := createAsanaService(t)

	accounts, err := s.Accounts()
	if err != nil {
		t.Error("error calling accounts(), err:", err)
	}

	if len(accounts) != 1 {
		t.Error("should get 1 account returned")
	}
	if accounts[0].ID != numberStrToInt64(os.Getenv("ASANA_ACCOUNT_ID")) {
		t.Error("got wrong account id")
	}
}

func TestIntegration_AsanaUsers(t *testing.T) {
	s := createAsanaService(t)

	users, err := s.Users()
	if err != nil {
		t.Error("error calling users(), err:", err)
	}

	if len(users) == 0 {
		t.Error("should get some users")
	}
}

func TestIntegration_AsanaProjects(t *testing.T) {
	defer resetAsanaLimit()
	asanaPerPageLimit = 10

	s := createAsanaService(t)

	projects, err := s.Projects()
	if err != nil {
		t.Error("error calling projects(), err:", err)
	}

	if len(projects) <= 10 {
		t.Error("should get more than 10 project, please create at least 11 project to test pagination")
	}
}

func TestIntegration_AsanaTask(t *testing.T) {
	defer resetAsanaLimit()
	asanaPerPageLimit = 10

	s := createAsanaService(t)

	tasks, err := s.Tasks()
	if err != nil {
		t.Error("error calling tasks(), err: ", err)
	}

	if len(tasks) <= 10 {
		t.Error("should get more than 10 tasks, please create at least 11 tasks and assign them to a project to test pagination")
	}
}

func resetAsanaLimit() {
	asanaPerPageLimit = 100
}

func createAsanaService(t *testing.T) *AsanaPipeIntegration {
	testToken := os.Getenv("ASANA_PERSONAL_TOKEN")
	testAccountID := os.Getenv("ASANA_ACCOUNT_ID")

	if testToken == "" || testAccountID == "" {
		t.Skipf("Skipped, because required environment variables (ASANA_PERSONAL_TOKEN, ASANA_ACCOUNT_ID) haven't been set.")
	}

	s := &AsanaPipeIntegration{}
	s.token = oauth.Token{
		AccessToken: testToken,
	}
	s.AsanaParams = &AsanaParams{
		AccountID: numberStrToInt64(testAccountID),
	}

	return s
}
