package connection

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
	_, err := ts.db.Exec(truncateConnectionSQL)
	ts.NoError(err)
}

func (ts *StorageTestSuite) TestStorage_Save_Load_Ok() {
	s := NewStorage(ts.db)
	c := NewConnection(1, "test1")

	err := s.Save(c)
	ts.NoError(err)

	cFromDb, err := s.Load(1, "test1")
	ts.NoError(err)
	ts.Equal(c, cFromDb)
}

func (ts *StorageTestSuite) TestStorage_Save_Load_DbClosed() {
	cdb, err := sql.Open("postgres", getDbConnString())
	require.NoError(ts.T(), err)
	cdb.Close()

	s := NewStorage(cdb)
	c := NewConnection(2, "test2")

	err = s.Save(c)
	ts.Error(err)

	con, err := s.Load(2, "test2")
	ts.Error(err)
	ts.Nil(con)
}

func (ts *StorageTestSuite) TestStorage_Save_LoadReversed_Ok() {
	s := NewStorage(ts.db)
	c := NewConnection(3, "test3")
	c.Data["1-test"] = 10
	c.Data["2-test"] = 20

	err := s.Save(c)
	ts.NoError(err)

	cFromDb, err := s.LoadReversed(3, "test3")
	ts.NoError(err)
	ts.Contains(cFromDb.GetKeys(), 10)
	ts.Contains(cFromDb.GetKeys(), 20)

	ts.Equal(1, cFromDb.GetForeignID(10))
	ts.Equal(2, cFromDb.GetForeignID(20))
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestStorageTestSuite(t *testing.T) {
	suite.Run(t, new(StorageTestSuite))
}
