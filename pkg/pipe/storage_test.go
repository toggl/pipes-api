package pipe

import (
	"database/sql"
	"flag"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
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
}

func (ts *StorageTestSuite) TestStorage_Save_Load_Ok() {
	s := NewStorage(ts.db)
	a := NewAuthorization(1, "github")

	err := s.SaveAuthorization(a)
	ts.NoError(err)

	aFromDb, err := s.LoadAuthorization(1, "github")
	ts.NoError(err)
	ts.Equal(a, aFromDb)
}

func (ts *StorageTestSuite) TestStorage_Save_Load_DbClosed() {
	cdb, err := sql.Open("postgres", getDbConnString())
	require.NoError(ts.T(), err)
	cdb.Close()

	s := NewStorage(cdb)

	a := NewAuthorization(2, "asana")
	err = s.SaveAuthorization(a)
	ts.Error(err)

	con, err := s.LoadAuthorization(2, "asana")
	ts.Error(err)
	ts.Nil(con)
}

func (ts *StorageTestSuite) TestStorage_Save_Destroy_Ok() {
	s := NewStorage(ts.db)

	a := NewAuthorization(1, "github")

	err := s.SaveAuthorization(a)
	ts.NoError(err)

	err = s.DestroyAuthorization(1, "github")
	ts.NoError(err)
}

func (ts *StorageTestSuite) TestStorage_Save_LoadWorkspaceAuthorizations_Ok() {
	s := NewStorage(ts.db)

	a1 := NewAuthorization(1, "github")
	a2 := NewAuthorization(1, "asana")

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
