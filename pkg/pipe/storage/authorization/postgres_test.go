package authorization

import (
	"database/sql"
	"flag"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/toggl/pipes-api/pkg/integration"
	"github.com/toggl/pipes-api/pkg/pipe"
	"github.com/toggl/pipes-api/pkg/pipe/mocks"
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

type AuthorizationsStorageTestSuite struct {
	suite.Suite
	db *sql.DB
}

func (ts *AuthorizationsStorageTestSuite) SetupSuite() {
	var err error
	ts.db, err = sql.Open("postgres", getDbConnString())
	require.NoError(ts.T(), err)

	err = ts.db.Ping()
	if err != nil {
		ts.T().Skipf("Could not connect to database, db_conn_string: %v", getDbConnString())
	}
}

func (ts *AuthorizationsStorageTestSuite) TearDownSuite() {
	ts.db.Close()
}

func (ts *AuthorizationsStorageTestSuite) SetupTest() {
	_, err1 := ts.db.Exec(truncateAuthorizationSQL)
	ts.NoError(err1)
}

func (ts *AuthorizationsStorageTestSuite) TestStorage_SaveAuthorization_LoadAuthorization_Ok() {

	af := &pipe.AuthorizationFactory{
		IntegrationsStorage:   &mocks.IntegrationsStorage{},
		AuthorizationsStorage: &mocks.AuthorizationsStorage{},
		OAuthProvider:         &mocks.OAuthProvider{},
	}
	a := af.Create(1, integration.GitHub)

	s := NewPostgresStorage(ts.db)
	err := s.Save(a)
	ts.NoError(err)

	aFromDb := af.Create(0, integration.GitHub)
	err = s.Load(1, integration.GitHub, aFromDb)
	ts.NoError(err)
	ts.Equal(a, aFromDb)
}

func (ts *AuthorizationsStorageTestSuite) TestStorage_SaveAuthorization_LoadAuthorization_DbClosed() {
	cdb, err := sql.Open("postgres", getDbConnString())
	require.NoError(ts.T(), err)
	cdb.Close()

	s := NewPostgresStorage(cdb)

	af := &pipe.AuthorizationFactory{
		IntegrationsStorage:   &mocks.IntegrationsStorage{},
		AuthorizationsStorage: &mocks.AuthorizationsStorage{},
		OAuthProvider:         &mocks.OAuthProvider{},
	}
	a := af.Create(2, integration.Asana)
	err = s.Save(a)
	ts.Error(err)

	err = s.Load(2, integration.Asana, a)
	ts.Error(err)
}

func (ts *AuthorizationsStorageTestSuite) TestStorage_SaveAuthorization_DestroyAuthorization_Ok() {
	s := NewPostgresStorage(ts.db)

	af := &pipe.AuthorizationFactory{
		IntegrationsStorage:   &mocks.IntegrationsStorage{},
		AuthorizationsStorage: &mocks.AuthorizationsStorage{},
		OAuthProvider:         &mocks.OAuthProvider{},
	}
	a := af.Create(1, integration.GitHub)

	err := s.Save(a)
	ts.NoError(err)

	err = s.Delete(1, integration.GitHub)
	ts.NoError(err)
}

func (ts *AuthorizationsStorageTestSuite) TestStorage_SaveAuthorization_LoadWorkspaceAuthorizations_Ok() {
	s := NewPostgresStorage(ts.db)

	af := &pipe.AuthorizationFactory{
		IntegrationsStorage:   &mocks.IntegrationsStorage{},
		AuthorizationsStorage: &mocks.AuthorizationsStorage{},
		OAuthProvider:         &mocks.OAuthProvider{},
	}

	a1 := af.Create(1, integration.GitHub)
	a2 := af.Create(1, integration.Asana)

	err := s.Save(a1)
	ts.NoError(err)

	err = s.Save(a2)
	ts.NoError(err)

	auth, err := s.LoadWorkspaceAuthorizations(1)
	ts.NoError(err)
	ts.Equal(true, auth[integration.GitHub])
	ts.Equal(true, auth[integration.Asana])
	ts.Equal(false, auth["unknown"])
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestAuthorizationsStorageTestSuite(t *testing.T) {
	suite.Run(t, new(AuthorizationsStorageTestSuite))
}
