package storage

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/toggl/pipes-api/pkg/domain"
	"github.com/toggl/pipes-api/pkg/domain/mocks"
	"github.com/toggl/pipes-api/pkg/integration"
)

type AuthorizationsStorageTestSuite struct {
	suite.Suite
	db *sql.DB
}

func (ts *AuthorizationsStorageTestSuite) SetupSuite() {
	var err error
	ts.db, err = sql.Open("postgres", getConnectionStringForTests())
	require.NoError(ts.T(), err)

	err = ts.db.Ping()
	if err != nil {
		ts.T().Skipf("Could not connect to database, db_conn_string: %v", getConnectionStringForTests())
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

	af := &domain.AuthorizationFactory{
		IntegrationsStorage:   &mocks.IntegrationsStorage{},
		AuthorizationsStorage: &mocks.AuthorizationsStorage{},
		OAuthProvider:         &mocks.OAuthProvider{},
	}
	a := af.Create(1, integration.GitHub)

	s := NewAuthorizationStorage(ts.db)
	err := s.Save(a)
	ts.NoError(err)

	aFromDb := af.Create(0, integration.GitHub)
	err = s.Load(1, integration.GitHub, aFromDb)
	ts.NoError(err)
	ts.Equal(a, aFromDb)
}

func (ts *AuthorizationsStorageTestSuite) TestStorage_SaveAuthorization_LoadAuthorization_DbClosed() {
	cdb, err := sql.Open("postgres", getConnectionStringForTests())
	require.NoError(ts.T(), err)
	cdb.Close()

	s := NewAuthorizationStorage(cdb)

	af := &domain.AuthorizationFactory{
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
	s := NewAuthorizationStorage(ts.db)

	af := &domain.AuthorizationFactory{
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
	s := NewAuthorizationStorage(ts.db)

	af := &domain.AuthorizationFactory{
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
