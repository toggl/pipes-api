package storage

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/toggl/pipes-api/internal/service"
	"github.com/toggl/pipes-api/pkg/domain"
)

type ImportsStorageTestSuite struct {
	suite.Suite
	db *sql.DB
}

func (ts *ImportsStorageTestSuite) SetupSuite() {
	var err error
	ts.db, err = sql.Open("postgres", getConnectionStringForTests())
	require.NoError(ts.T(), err)

	err = ts.db.Ping()
	if err != nil {
		ts.T().Skipf("Could not connect to database, db_conn_string: %v", getConnectionStringForTests())
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
	s := &ImportStorage{db: ts.db}
	svc := service.NewPipeIntegration(domain.GitHub, 1)
	err := s.DeleteAccountsFor(svc)
	ts.NoError(err)
}

func (ts *ImportsStorageTestSuite) TestStorage_DeleteUsersFor() {
	s := &ImportStorage{db: ts.db}
	svc := service.NewPipeIntegration(domain.GitHub, 1)
	err := s.DeleteUsersFor(svc)
	ts.NoError(err)
}

func (ts *ImportsStorageTestSuite) TestStorage_SaveAccountsFor_LoadAccountsFor() {
	s := &ImportStorage{db: ts.db}
	svc := service.NewPipeIntegration(domain.GitHub, 1)

	resp := &domain.AccountsResponse{
		Error: "",
		Accounts: []*domain.Account{
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
	s := &ImportStorage{db: ts.db}
	svc := service.NewPipeIntegration(domain.GitHub, 1)

	resp := &domain.UsersResponse{
		Error: "",
		Users: []*domain.User{
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
	s := &ImportStorage{db: ts.db}
	svc := service.NewPipeIntegration(domain.GitHub, 1)

	resp := &domain.ClientsResponse{
		Error: "",
		Clients: []*domain.Client{
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
	s := &ImportStorage{db: ts.db}
	svc := service.NewPipeIntegration(domain.GitHub, 1)

	resp := &domain.ProjectsResponse{
		Error: "",
		Projects: []*domain.Project{
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
	s := &ImportStorage{db: ts.db}
	svc := service.NewPipeIntegration(domain.GitHub, 1)

	resp := &domain.TasksResponse{
		Error: "",
		Tasks: []*domain.Task{
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
	s := &ImportStorage{db: ts.db}
	svc := service.NewPipeIntegration(domain.GitHub, 1)

	resp := &domain.TasksResponse{
		Error: "",
		Tasks: []*domain.Task{
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
