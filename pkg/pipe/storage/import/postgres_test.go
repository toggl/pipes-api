package _import

import (
	"database/sql"
	"flag"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/toggl/pipes-api/pkg/integration"
	"github.com/toggl/pipes-api/pkg/pipe"
	"github.com/toggl/pipes-api/pkg/toggl"
)

var (
	dbConnString string
	mx           sync.RWMutex
)

func getDbConnString() string {
	mx.RLock()
	defer mx.RUnlock()
	return dbConnString
}

func init() {
	// There is no need to call "flag.Parse()". See: https://golang.org/doc/go1.13#testing
	flag.StringVar(&dbConnString, "db_conn_string", "dbname=pipes_test user=pipes_user host=localhost sslmode=disable port=5432", "Database Connection String")
}

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
	_, err5 := ts.db.Exec(truncateImportsSQL)
	ts.NoError(err5)
}

func (ts *ImportsStorageTestSuite) TestStorage_DeleteAccountsFor() {
	s := NewPostgresStorage(ts.db)
	svc := pipe.NewExternalService(integration.GitHub, 1)
	err := s.DeleteAccountsFor(svc)
	ts.NoError(err)
}

func (ts *ImportsStorageTestSuite) TestStorage_DeleteUsersFor() {
	s := NewPostgresStorage(ts.db)
	svc := pipe.NewExternalService(integration.GitHub, 1)
	err := s.DeleteUsersFor(svc)
	ts.NoError(err)
}

func (ts *ImportsStorageTestSuite) TestStorage_SaveAccountsFor_LoadAccountsFor() {
	s := NewPostgresStorage(ts.db)
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
	s := NewPostgresStorage(ts.db)
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
	s := NewPostgresStorage(ts.db)
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
	s := NewPostgresStorage(ts.db)
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
	s := NewPostgresStorage(ts.db)
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
	s := NewPostgresStorage(ts.db)
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
