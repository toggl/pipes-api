package storage

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/toggl/pipes-api/pkg/integration"
	"github.com/toggl/pipes-api/pkg/pipe"
	"github.com/toggl/pipes-api/pkg/toggl"
)

type ImportsStorageTestSuite struct {
	suite.Suite
	db *sql.DB
}

func (ts *ImportsStorageTestSuite) SetupSuite() {
	var err error
	ts.db, err = sql.Open("postgres", getDbConnString())
	require.NoError(ts.T(), err)

	err = ts.db.Ping()
	if err != nil {
		ts.T().Skipf("Could not connect to database, db_conn_string: %v", getDbConnString())
	}
}

func (ts *ImportsStorageTestSuite) TearDownSuite() {
	ts.db.Close()
}

func (ts *ImportsStorageTestSuite) SetupTest() {
	_, err1 := ts.db.Exec(truncateAuthorizationSQL)
	_, err2 := ts.db.Exec(truncateConnectionSQL)
	_, err3 := ts.db.Exec(truncatePipesStatusSQL)
	_, err4 := ts.db.Exec(truncatePipesSQL)
	_, err5 := ts.db.Exec(truncateImportsSQL)

	ts.NoError(err1)
	ts.NoError(err2)
	ts.NoError(err3)
	ts.NoError(err4)
	ts.NoError(err5)
}

func (ts *ImportsStorageTestSuite) TestStorage_DeleteAccountsFor() {
	s := NewImportsPostgresStorage(ts.db)
	svc := pipe.NewExternalService(integration.GitHub, 1)
	err := s.DeleteAccountsFor(svc)
	ts.NoError(err)
}

func (ts *ImportsStorageTestSuite) TestStorage_DeleteUsersFor() {
	s := NewImportsPostgresStorage(ts.db)
	svc := pipe.NewExternalService(integration.GitHub, 1)
	err := s.DeleteUsersFor(svc)
	ts.NoError(err)
}

func (ts *ImportsStorageTestSuite) TestStorage_SaveAccountsFor_LoadAccountsFor() {
	s := NewImportsPostgresStorage(ts.db)
	svc := pipe.NewExternalService(integration.GitHub, 1)

	resp := &toggl.AccountsResponse{
		Error: "",
		Accounts: []*toggl.Account{
			{ID: 1, Name: "test1"},
			{ID: 2, Name: "test2"},
		},
	}

	err := s.SaveAccountsFor(svc, *resp)
	ts.NoError(err)

	got, err := s.LoadAccountsFor(svc)
	ts.NoError(err)
	ts.Equal(resp, got)
}

func (ts *ImportsStorageTestSuite) TestStorage_SaveUsersFor_LoadUsersFor() {
	s := NewImportsPostgresStorage(ts.db)
	svc := pipe.NewExternalService(integration.GitHub, 1)

	resp := &toggl.UsersResponse{
		Error: "",
		Users: []*toggl.User{
			{ID: 1, Name: "test1"},
			{ID: 2, Name: "test2"},
		},
	}

	err := s.SaveUsersFor(svc, *resp)
	ts.NoError(err)

	got, err := s.LoadUsersFor(svc)
	ts.NoError(err)
	ts.Equal(resp, got)
}

func (ts *ImportsStorageTestSuite) TestStorage_SaveClientsFor_LoadClientsFor() {
	s := NewImportsPostgresStorage(ts.db)
	svc := pipe.NewExternalService(integration.GitHub, 1)

	resp := &toggl.ClientsResponse{
		Error: "",
		Clients: []*toggl.Client{
			{ID: 1, Name: "test1"},
			{ID: 2, Name: "test2"},
		},
	}

	err := s.SaveClientsFor(svc, *resp)
	ts.NoError(err)

	got, err := s.LoadClientsFor(svc)
	ts.NoError(err)
	ts.Equal(resp, got)
}

func (ts *ImportsStorageTestSuite) TestStorage_SaveProjectsFor_LoadProjectsFor() {
	s := NewImportsPostgresStorage(ts.db)
	svc := pipe.NewExternalService(integration.GitHub, 1)

	resp := &toggl.ProjectsResponse{
		Error: "",
		Projects: []*toggl.Project{
			{
				ID:       1,
				Name:     "test1",
				Active:   true,
				Billable: false,
				ClientID: 3,
			},
			{
				ID:       2,
				Name:     "test2",
				Active:   false,
				Billable: true,
				ClientID: 4,
			},
		},
	}

	err := s.SaveProjectsFor(svc, *resp)
	ts.NoError(err)

	got, err := s.LoadProjectsFor(svc)
	ts.NoError(err)
	ts.Equal(resp, got)
}

func (ts *ImportsStorageTestSuite) TestStorage_SaveTasksFor_LoadTasksFor() {
	s := NewImportsPostgresStorage(ts.db)
	svc := pipe.NewExternalService(integration.GitHub, 1)

	resp := &toggl.TasksResponse{
		Error: "",
		Tasks: []*toggl.Task{
			{
				ID:        1,
				Name:      "test1",
				Active:    true,
				ProjectID: 3,
			},
			{
				ID:        2,
				Name:      "test2",
				Active:    false,
				ProjectID: 4,
			},
		},
	}

	err := s.SaveTasksFor(svc, *resp)
	ts.NoError(err)

	got, err := s.LoadTasksFor(svc)
	ts.NoError(err)
	ts.Equal(resp, got)
}

func (ts *ImportsStorageTestSuite) TestStorage_SaveTodoListsFor_LoadTodoListsFor() {
	s := NewImportsPostgresStorage(ts.db)
	svc := pipe.NewExternalService(integration.GitHub, 1)

	resp := &toggl.TasksResponse{
		Error: "",
		Tasks: []*toggl.Task{
			{
				ID:        1,
				Name:      "test1",
				Active:    true,
				ProjectID: 3,
			},
			{
				ID:        2,
				Name:      "test2",
				Active:    false,
				ProjectID: 4,
			},
		},
	}

	err := s.SaveTodoListsFor(svc, *resp)
	ts.NoError(err)

	got, err := s.LoadTodoListsFor(svc)
	ts.NoError(err)
	ts.Equal(resp, got)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestImportsStorageTestSuite(t *testing.T) {
	suite.Run(t, new(ImportsStorageTestSuite))
}
