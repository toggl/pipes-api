package storage

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/toggl/pipes-api/internal/service"
	"github.com/toggl/pipes-api/pkg/domain"
)

type IDMappingsStorageTestSuite struct {
	suite.Suite
	db *sql.DB
}

func (ts *IDMappingsStorageTestSuite) SetupSuite() {
	var err error
	ts.db, err = sql.Open("postgres", getConnectionStringForTests())
	require.NoError(ts.T(), err)

	err = ts.db.Ping()
	if err != nil {
		ts.T().Skipf("Could not connect to database, db_conn_string: %v", getConnectionStringForTests())
	}
}

func (ts *IDMappingsStorageTestSuite) TearDownSuite() {
	ts.db.Close()
}

func (ts *IDMappingsStorageTestSuite) SetupTest() {
	_, err2 := ts.db.Exec(truncateConnectionSQL)
	_, err3 := ts.db.Exec(truncatePipesStatusSQL)
	ts.NoError(err2)
	ts.NoError(err3)
}

func (ts *IDMappingsStorageTestSuite) TestStorage_SaveConnection_LoadConnection_Ok() {
	s := &IdMappingStorage{DB: ts.db}
	c := domain.NewIDMapping(1, "test1")

	err := s.Save(c)
	ts.NoError(err)

	cFromDb, err := s.Load(1, "test1")
	ts.NoError(err)
	ts.Equal(c, cFromDb)
}

func (ts *IDMappingsStorageTestSuite) TestStorage_SaveConnection_LoadConnection_DbClosed() {
	cdb, err := sql.Open("postgres", getConnectionStringForTests())
	require.NoError(ts.T(), err)
	cdb.Close()

	s := &IdMappingStorage{DB: cdb}
	c := domain.NewIDMapping(2, "test2")

	err = s.Save(c)
	ts.Error(err)

	con, err := s.Load(2, "test2")
	ts.Error(err)
	ts.Nil(con)
}

func (ts *IDMappingsStorageTestSuite) TestStorage_SaveConnection_LoadReversedConnection_Ok() {
	s := &IdMappingStorage{DB: ts.db}
	c := domain.NewIDMapping(3, "test3")
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

func (ts *IDMappingsStorageTestSuite) TestStorage_DeletePipeConnections() {
	s := &IdMappingStorage{DB: ts.db}

	p1 := createPipeForTests(1, domain.GitHub, domain.UsersPipe)
	p1.PipeStatus = domain.NewPipeStatus(1, domain.GitHub, domain.UsersPipe, "test")
	svc := service.NewExternalService(domain.GitHub, 1)

	err := s.Delete(1, svc.KeyFor(p1.ID), p1.PipeStatus.Key)
	ts.NoError(err)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestIDMappingsStorageTestSuite(t *testing.T) {
	suite.Run(t, new(IDMappingsStorageTestSuite))
}
