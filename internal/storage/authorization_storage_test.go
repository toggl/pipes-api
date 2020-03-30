package storage

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/toggl/pipes-api/pkg/domain"
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

	a := domain.NewAuthorization(1, domain.GitHub)

	s := &AuthorizationStorage{db: ts.db}
	err := s.Save(a)
	ts.NoError(err)

	aFromDb := domain.NewAuthorization(0, domain.GitHub)
	err = s.Load(1, domain.GitHub, aFromDb)
	ts.NoError(err)
	ts.Equal(a, aFromDb)
}

func (ts *AuthorizationsStorageTestSuite) TestStorage_SaveAuthorization_LoadAuthorization_DbClosed() {
	cdb, err := sql.Open("postgres", getConnectionStringForTests())
	require.NoError(ts.T(), err)
	cdb.Close()

	s := &AuthorizationStorage{db: cdb}
	a := domain.NewAuthorization(2, domain.Asana)
	err = s.Save(a)
	ts.Error(err)

	err = s.Load(2, domain.Asana, a)
	ts.Error(err)
}

func (ts *AuthorizationsStorageTestSuite) TestStorage_SaveAuthorization_DestroyAuthorization_Ok() {
	s := &AuthorizationStorage{db: ts.db}
	a := domain.NewAuthorization(1, domain.GitHub)

	err := s.Save(a)
	ts.NoError(err)

	err = s.Delete(1, domain.GitHub)
	ts.NoError(err)
}

func (ts *AuthorizationsStorageTestSuite) TestStorage_SaveAuthorization_LoadWorkspaceAuthorizations_Ok() {
	s := &AuthorizationStorage{db: ts.db}

	a1 := domain.NewAuthorization(1, domain.GitHub)
	a2 := domain.NewAuthorization(1, domain.Asana)

	err := s.Save(a1)
	ts.NoError(err)

	err = s.Save(a2)
	ts.NoError(err)

	auth, err := s.LoadWorkspaceAuthorizations(1)
	ts.NoError(err)
	ts.Equal(true, auth[domain.GitHub])
	ts.Equal(true, auth[domain.Asana])
	ts.Equal(false, auth["unknown"])
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestAuthorizationsStorageTestSuite(t *testing.T) {
	suite.Run(t, new(AuthorizationsStorageTestSuite))
}