package storage

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/toggl/pipes-api/pkg/integration"
	"github.com/toggl/pipes-api/pkg/pipe"
)

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
		IntegrationsStorage:   &pipe.MockIntegrationsStorage{},
		AuthorizationsStorage: &pipe.MockAuthorizationsStorage{},
		OAuthProvider:         &pipe.MockOAuthProvider{},
	}
	a := af.Create(1, integration.GitHub)

	s := NewAuthorizationsPostgresStorage(ts.db)
	err := s.SaveAuthorization(a)
	ts.NoError(err)

	aFromDb := af.Create(0, integration.GitHub)
	err = s.LoadAuthorization(1, integration.GitHub, aFromDb)
	ts.NoError(err)
	ts.Equal(a, aFromDb)
}

func (ts *AuthorizationsStorageTestSuite) TestStorage_SaveAuthorization_LoadAuthorization_DbClosed() {
	cdb, err := sql.Open("postgres", getDbConnString())
	require.NoError(ts.T(), err)
	cdb.Close()

	s := NewAuthorizationsPostgresStorage(cdb)

	af := &pipe.AuthorizationFactory{
		IntegrationsStorage:   &pipe.MockIntegrationsStorage{},
		AuthorizationsStorage: &pipe.MockAuthorizationsStorage{},
		OAuthProvider:         &pipe.MockOAuthProvider{},
	}
	a := af.Create(2, integration.Asana)
	err = s.SaveAuthorization(a)
	ts.Error(err)

	err = s.LoadAuthorization(2, integration.Asana, a)
	ts.Error(err)
}

func (ts *AuthorizationsStorageTestSuite) TestStorage_SaveAuthorization_DestroyAuthorization_Ok() {
	s := NewAuthorizationsPostgresStorage(ts.db)

	af := &pipe.AuthorizationFactory{
		IntegrationsStorage:   &pipe.MockIntegrationsStorage{},
		AuthorizationsStorage: &pipe.MockAuthorizationsStorage{},
		OAuthProvider:         &pipe.MockOAuthProvider{},
	}
	a := af.Create(1, integration.GitHub)

	err := s.SaveAuthorization(a)
	ts.NoError(err)

	err = s.DeleteAuthorization(1, integration.GitHub)
	ts.NoError(err)
}

func (ts *AuthorizationsStorageTestSuite) TestStorage_SaveAuthorization_LoadWorkspaceAuthorizations_Ok() {
	s := NewAuthorizationsPostgresStorage(ts.db)

	af := &pipe.AuthorizationFactory{
		IntegrationsStorage:   &pipe.MockIntegrationsStorage{},
		AuthorizationsStorage: &pipe.MockAuthorizationsStorage{},
		OAuthProvider:         &pipe.MockOAuthProvider{},
	}

	a1 := af.Create(1, integration.GitHub)
	a2 := af.Create(1, integration.Asana)

	err := s.SaveAuthorization(a1)
	ts.NoError(err)

	err = s.SaveAuthorization(a2)
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
