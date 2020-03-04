package storage

import (
	"database/sql"
	"flag"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/toggl/pipes-api/pkg/pipe"
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

type StorageTestSuite struct {
	suite.Suite
	db *sql.DB
}

func (ts *StorageTestSuite) SetupSuite() {
	var err error
	ts.db, err = sql.Open("postgres", getDbConnString())
	require.NoError(ts.T(), err)

	err = ts.db.Ping()
	if err != nil {
		ts.T().Skipf("Could not connect to database, db_conn_string: %v", getDbConnString())
	}
}

func (ts *StorageTestSuite) TearDownSuite() {
	ts.db.Close()
}

func (ts *StorageTestSuite) SetupTest() {
	_, err := ts.db.Exec(truncateAuthorizationSQL)
	ts.NoError(err)
	_, err = ts.db.Exec(truncateConnectionSQL)
	ts.NoError(err)
}

func (ts *StorageTestSuite) TestStorage_SaveConnection_LoadConnection_Ok() {
	s := NewPostgresStorage(ts.db)
	c := pipe.NewConnection(1, "test1")

	err := s.SaveConnection(c)
	ts.NoError(err)

	cFromDb, err := s.LoadConnection(1, "test1")
	ts.NoError(err)
	ts.Equal(c, cFromDb)
}

func (ts *StorageTestSuite) TestStorage_SaveConnection_LoadConnection_DbClosed() {
	cdb, err := sql.Open("postgres", getDbConnString())
	require.NoError(ts.T(), err)
	cdb.Close()

	s := NewPostgresStorage(cdb)
	c := pipe.NewConnection(2, "test2")

	err = s.SaveConnection(c)
	ts.Error(err)

	con, err := s.LoadConnection(2, "test2")
	ts.Error(err)
	ts.Nil(con)
}

func (ts *StorageTestSuite) TestStorage_SaveConnection_LoadReversedConnection_Ok() {
	s := NewPostgresStorage(ts.db)
	c := pipe.NewConnection(3, "test3")
	c.Data["1-test"] = 10
	c.Data["2-test"] = 20

	err := s.SaveConnection(c)
	ts.NoError(err)

	cFromDb, err := s.LoadReversedConnection(3, "test3")
	ts.NoError(err)
	ts.Contains(cFromDb.GetKeys(), 10)
	ts.Contains(cFromDb.GetKeys(), 20)

	ts.Equal(1, cFromDb.GetForeignID(10))
	ts.Equal(2, cFromDb.GetForeignID(20))
}

func (ts *StorageTestSuite) TestStorage_SaveAuthorization_LoadAuthorization_Ok() {
	s := NewPostgresStorage(ts.db)
	a := pipe.NewAuthorization(1, "github")

	err := s.SaveAuthorization(a)
	ts.NoError(err)

	aFromDb, err := s.LoadAuthorization(1, "github")
	ts.NoError(err)
	ts.Equal(a, aFromDb)
}

func (ts *StorageTestSuite) TestStorage_SaveAuthorization_LoadAuthorization_DbClosed() {
	cdb, err := sql.Open("postgres", getDbConnString())
	require.NoError(ts.T(), err)
	cdb.Close()

	s := NewPostgresStorage(cdb)

	a := pipe.NewAuthorization(2, "asana")
	err = s.SaveAuthorization(a)
	ts.Error(err)

	con, err := s.LoadAuthorization(2, "asana")
	ts.Error(err)
	ts.Nil(con)
}

func (ts *StorageTestSuite) TestStorage_SaveAuthorization_DestroyAuthorization_Ok() {
	s := NewPostgresStorage(ts.db)

	a := pipe.NewAuthorization(1, "github")

	err := s.SaveAuthorization(a)
	ts.NoError(err)

	err = s.DestroyAuthorization(1, "github")
	ts.NoError(err)
}

func (ts *StorageTestSuite) TestStorage_SaveAuthorization_LoadWorkspaceAuthorizations_Ok() {
	s := NewPostgresStorage(ts.db)

	a1 := pipe.NewAuthorization(1, "github")
	a2 := pipe.NewAuthorization(1, "asana")

	err := s.SaveAuthorization(a1)
	ts.NoError(err)

	err = s.SaveAuthorization(a2)
	ts.NoError(err)

	auth, err := s.LoadWorkspaceAuthorizations(1)
	ts.NoError(err)
	ts.Equal(true, auth["github"])
	ts.Equal(true, auth["asana"])
	ts.Equal(false, auth["unknown"])
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestStorageTestSuite(t *testing.T) {
	suite.Run(t, new(StorageTestSuite))
}
