package pipe

import (
	"database/sql"
	"flag"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/toggl/pipes-api/pkg/integration"
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
	_, err3 := ts.db.Exec(truncatePipesStatusSQL)
	_, err4 := ts.db.Exec(truncatePipesSQL)

	ts.NoError(err3)
	ts.NoError(err4)
}

func (ts *StorageTestSuite) TestStorage_IsDown() {
	cdb, err := sql.Open("postgres", getDbConnString())
	require.NoError(ts.T(), err)

	s := NewPostgresStorage(cdb)
	ts.False(s.IsDown())

	cdb.Close()
	ts.True(s.IsDown())
}

func (ts *StorageTestSuite) TestStorage_Save_Load() {
	s := NewPostgresStorage(ts.db)

	p1 := pipe.NewPipe(1, integration.GitHub, integration.UsersPipe)
	err := s.Save(p1)
	ts.NoError(err)

	p2, err := s.Load(1, integration.GitHub, integration.UsersPipe)
	ts.NoError(err)
	ts.Equal(p1, p2)
}

func (ts *StorageTestSuite) TestStorage_SavePipeStatus_LoadPipeStatus() {
	s := NewPostgresStorage(ts.db)

	p1 := pipe.NewPipeStatus(1, integration.GitHub, integration.UsersPipe, "")
	p1.Status = pipe.StatusSuccess
	p1.ObjectCounts = []string{"obj1", "obj2"}

	err := s.SaveStatus(p1)
	ts.NoError(err)

	p2, err := s.LoadStatus(1, integration.GitHub, integration.UsersPipe)
	ts.NoError(err)
	ts.Equal(p1.WorkspaceID, p2.WorkspaceID)
	ts.Equal(p1.ServiceID, p2.ServiceID)
	ts.Equal(p1.PipeID, p2.PipeID)
	ts.Contains(p2.Message, "successfully imported/exported")

	p3 := pipe.NewPipeStatus(2, integration.GitHub, integration.UsersPipe, "")
	p3.Status = pipe.StatusSuccess
	err = s.SaveStatus(p3)
	ts.NoError(err)

	p4, err := s.LoadStatus(2, integration.GitHub, integration.UsersPipe)
	ts.NoError(err)

	ts.Contains(p4.Message, "No new")
}

func (ts *StorageTestSuite) TestStorage_SavePipeStatus_LoadPipeStatuses() {
	s := NewPostgresStorage(ts.db)

	p1 := pipe.NewPipeStatus(1, integration.GitHub, integration.UsersPipe, "")
	p2 := pipe.NewPipeStatus(1, integration.Asana, integration.UsersPipe, "")

	err := s.SaveStatus(p1)
	ts.NoError(err)
	err = s.SaveStatus(p2)
	ts.NoError(err)

	ss, err := s.LoadAllStatuses(1)
	ts.NoError(err)

	ts.Equal(2, len(ss))
}

func (ts *StorageTestSuite) TestStorage_Save_LoadPipes() {
	s := NewPostgresStorage(ts.db)

	p1 := pipe.NewPipe(1, integration.GitHub, integration.UsersPipe)
	err := s.Save(p1)
	ts.NoError(err)

	p2 := pipe.NewPipe(1, integration.Asana, integration.UsersPipe)
	err = s.Save(p2)
	ts.NoError(err)

	ps, err := s.LoadAll(1)
	ts.NoError(err)
	ts.Equal(2, len(ps))
}

func (ts *StorageTestSuite) TestStorage_Save_Delete() {
	s := NewPostgresStorage(ts.db)

	p1 := pipe.NewPipe(1, integration.GitHub, integration.UsersPipe)
	err := s.Save(p1)
	ts.NoError(err)

	p2 := pipe.NewPipe(1, integration.Asana, integration.UsersPipe)
	err = s.Save(p2)
	ts.NoError(err)

	ps, err := s.LoadAll(1)
	ts.NoError(err)
	ts.Equal(2, len(ps))

	err = s.Delete(p1, 1)
	ts.NoError(err)

	ps, err = s.LoadAll(1)
	ts.NoError(err)
	ts.Equal(1, len(ps))
}

func (ts *StorageTestSuite) TestStorage_Save_DeletePipeByWorkspaceIDServiceID() {
	s := NewPostgresStorage(ts.db)

	p1 := pipe.NewPipe(1, integration.GitHub, integration.UsersPipe)
	err := s.Save(p1)
	ts.NoError(err)

	p2 := pipe.NewPipe(1, integration.GitHub, integration.ProjectsPipe)
	err = s.Save(p2)
	ts.NoError(err)

	ps, err := s.LoadAll(1)
	ts.NoError(err)
	ts.Equal(2, len(ps))

	err = s.DeleteByWorkspaceIDServiceID(1, integration.GitHub)
	ts.NoError(err)

	ps, err = s.LoadAll(1)
	ts.NoError(err)
	ts.Equal(0, len(ps))
}

func (ts *StorageTestSuite) TestStorage_Save_LoadLastSync() {
	s := NewPostgresStorage(ts.db)
	t := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)

	p1 := pipe.NewPipe(1, integration.GitHub, integration.UsersPipe)
	p1.ServiceParams = []byte(`{"start_date":"2020-01-02"}`)
	err := s.Save(p1)
	ts.NoError(err)

	s.LoadLastSyncFor(p1)
	ts.NotNil(p1.LastSync)
	ts.Equal(t, *p1.LastSync)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestStorageTestSuite(t *testing.T) {
	suite.Run(t, new(StorageTestSuite))
}
