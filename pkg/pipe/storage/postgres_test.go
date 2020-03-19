package storage

import (
	"database/sql"
	"encoding/json"
	"flag"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/toggl/pipes-api/pkg/integrations"
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
	_, err1 := ts.db.Exec(truncateAuthorizationSQL)
	_, err2 := ts.db.Exec(truncateConnectionSQL)
	_, err3 := ts.db.Exec(truncatePipesStatusSQL)
	_, err4 := ts.db.Exec(truncatePipesSQL)
	_, err5 := ts.db.Exec(truncateImportsSQL)
	_, err6 := ts.db.Exec(truncateQueuedPipesSQL)

	ts.NoError(err1)
	ts.NoError(err2)
	ts.NoError(err3)
	ts.NoError(err4)
	ts.NoError(err5)
	ts.NoError(err6)
}

func (ts *StorageTestSuite) TestStorage_SaveConnection_LoadConnection_Ok() {
	s := NewPostgresStorage(ts.db)
	c := pipe.NewIDMapping(1, "test1")

	err := s.SaveIDMapping(c)
	ts.NoError(err)

	cFromDb, err := s.LoadIDMapping(1, "test1")
	ts.NoError(err)
	ts.Equal(c, cFromDb)
}

func (ts *StorageTestSuite) TestStorage_SaveConnection_LoadConnection_DbClosed() {
	cdb, err := sql.Open("postgres", getDbConnString())
	require.NoError(ts.T(), err)
	cdb.Close()

	s := NewPostgresStorage(cdb)
	c := pipe.NewIDMapping(2, "test2")

	err = s.SaveIDMapping(c)
	ts.Error(err)

	con, err := s.LoadIDMapping(2, "test2")
	ts.Error(err)
	ts.Nil(con)
}

func (ts *StorageTestSuite) TestStorage_SaveConnection_LoadReversedConnection_Ok() {
	s := NewPostgresStorage(ts.db)
	c := pipe.NewIDMapping(3, "test3")
	c.Data["1-test"] = 10
	c.Data["2-test"] = 20

	err := s.SaveIDMapping(c)
	ts.NoError(err)

	cFromDb, err := s.LoadReversedIDMapping(3, "test3")
	ts.NoError(err)
	ts.Contains(cFromDb.GetKeys(), 10)
	ts.Contains(cFromDb.GetKeys(), 20)

	ts.Equal(1, cFromDb.GetForeignID(10))
	ts.Equal(2, cFromDb.GetForeignID(20))
}

func (ts *StorageTestSuite) TestStorage_SaveAuthorization_LoadAuthorization_Ok() {
	s := NewPostgresStorage(ts.db)
	a := pipe.NewAuthorization(1, integrations.GitHub)

	err := s.SaveAuthorization(a)
	ts.NoError(err)

	aFromDb, err := s.LoadAuthorization(1, integrations.GitHub)
	ts.NoError(err)
	ts.Equal(a, aFromDb)
}

func (ts *StorageTestSuite) TestStorage_SaveAuthorization_LoadAuthorization_DbClosed() {
	cdb, err := sql.Open("postgres", getDbConnString())
	require.NoError(ts.T(), err)
	cdb.Close()

	s := NewPostgresStorage(cdb)

	a := pipe.NewAuthorization(2, integrations.Asana)
	err = s.SaveAuthorization(a)
	ts.Error(err)

	con, err := s.LoadAuthorization(2, integrations.Asana)
	ts.Error(err)
	ts.Nil(con)
}

func (ts *StorageTestSuite) TestStorage_SaveAuthorization_DestroyAuthorization_Ok() {
	s := NewPostgresStorage(ts.db)

	a := pipe.NewAuthorization(1, integrations.GitHub)

	err := s.SaveAuthorization(a)
	ts.NoError(err)

	err = s.DeleteAuthorization(1, integrations.GitHub)
	ts.NoError(err)
}

func (ts *StorageTestSuite) TestStorage_SaveAuthorization_LoadWorkspaceAuthorizations_Ok() {
	s := NewPostgresStorage(ts.db)

	a1 := pipe.NewAuthorization(1, integrations.GitHub)
	a2 := pipe.NewAuthorization(1, integrations.Asana)

	err := s.SaveAuthorization(a1)
	ts.NoError(err)

	err = s.SaveAuthorization(a2)
	ts.NoError(err)

	auth, err := s.LoadWorkspaceAuthorizations(1)
	ts.NoError(err)
	ts.Equal(true, auth[integrations.GitHub])
	ts.Equal(true, auth[integrations.Asana])
	ts.Equal(false, auth["unknown"])
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

	p1 := pipe.NewPipe(1, integrations.GitHub, integrations.UsersPipe)
	err := s.Save(p1)
	ts.NoError(err)

	p2, err := s.LoadPipe(1, integrations.GitHub, integrations.UsersPipe)
	ts.NoError(err)
	ts.Equal(p1, p2)
}

func (ts *StorageTestSuite) TestStorage_SavePipeStatus_LoadPipeStatus() {
	s := NewPostgresStorage(ts.db)

	p1 := pipe.NewPipeStatus(1, integrations.GitHub, integrations.UsersPipe, "")
	p1.Status = pipe.StatusSuccess
	p1.ObjectCounts = []string{"obj1", "obj2"}

	err := s.SavePipeStatus(p1)
	ts.NoError(err)

	p2, err := s.LoadPipeStatus(1, integrations.GitHub, integrations.UsersPipe)
	ts.NoError(err)
	ts.Equal(p1.WorkspaceID, p2.WorkspaceID)
	ts.Equal(p1.ServiceID, p2.ServiceID)
	ts.Equal(p1.PipeID, p2.PipeID)
	ts.Contains(p2.Message, "successfully imported/exported")

	p3 := pipe.NewPipeStatus(2, integrations.GitHub, integrations.UsersPipe, "")
	p3.Status = pipe.StatusSuccess
	err = s.SavePipeStatus(p3)
	ts.NoError(err)

	p4, err := s.LoadPipeStatus(2, integrations.GitHub, integrations.UsersPipe)
	ts.NoError(err)

	ts.Contains(p4.Message, "No new")
}

func (ts *StorageTestSuite) TestStorage_SavePipeStatus_LoadPipeStatuses() {
	s := NewPostgresStorage(ts.db)

	p1 := pipe.NewPipeStatus(1, integrations.GitHub, integrations.UsersPipe, "")
	p2 := pipe.NewPipeStatus(1, integrations.Asana, integrations.UsersPipe, "")

	err := s.SavePipeStatus(p1)
	ts.NoError(err)
	err = s.SavePipeStatus(p2)
	ts.NoError(err)

	ss, err := s.LoadPipeStatuses(1)
	ts.NoError(err)

	ts.Equal(2, len(ss))
}

func (ts *StorageTestSuite) TestStorage_Save_LoadPipes() {
	s := NewPostgresStorage(ts.db)

	p1 := pipe.NewPipe(1, integrations.GitHub, integrations.UsersPipe)
	err := s.Save(p1)
	ts.NoError(err)

	p2 := pipe.NewPipe(1, integrations.Asana, integrations.UsersPipe)
	err = s.Save(p2)
	ts.NoError(err)

	ps, err := s.LoadPipes(1)
	ts.NoError(err)
	ts.Equal(2, len(ps))
}

func (ts *StorageTestSuite) TestStorage_SaveObject_LoadObject() {
	s := NewPostgresStorage(ts.db)

	type obj struct {
		Name  string
		Value string
	}
	o := obj{"Test", "Test2"}
	b1, err := json.Marshal(o)
	ts.NoError(err)

	svc := pipe.NewExternalService(integrations.GitHub, 1)

	err = s.saveObject(svc, integrations.ProjectsPipe, b1)
	ts.NoError(err)

	b, err := s.loadObject(svc, integrations.ProjectsPipe)
	ts.NoError(err)

	ts.Equal(`{"Name":"Test","Value":"Test2"}`, string(b))
}

func (ts *StorageTestSuite) TestStorage_Save_Delete() {
	s := NewPostgresStorage(ts.db)

	p1 := pipe.NewPipe(1, integrations.GitHub, integrations.UsersPipe)
	err := s.Save(p1)
	ts.NoError(err)

	p2 := pipe.NewPipe(1, integrations.Asana, integrations.UsersPipe)
	err = s.Save(p2)
	ts.NoError(err)

	ps, err := s.LoadPipes(1)
	ts.NoError(err)
	ts.Equal(2, len(ps))

	err = s.Delete(p1, 1)
	ts.NoError(err)

	ps, err = s.LoadPipes(1)
	ts.NoError(err)
	ts.Equal(1, len(ps))
}

func (ts *StorageTestSuite) TestStorage_Save_DeletePipeByWorkspaceIDServiceID() {
	s := NewPostgresStorage(ts.db)

	p1 := pipe.NewPipe(1, integrations.GitHub, integrations.UsersPipe)
	err := s.Save(p1)
	ts.NoError(err)

	p2 := pipe.NewPipe(1, integrations.GitHub, integrations.ProjectsPipe)
	err = s.Save(p2)
	ts.NoError(err)

	ps, err := s.LoadPipes(1)
	ts.NoError(err)
	ts.Equal(2, len(ps))

	err = s.DeletePipeByWorkspaceIDServiceID(1, integrations.GitHub)
	ts.NoError(err)

	ps, err = s.LoadPipes(1)
	ts.NoError(err)
	ts.Equal(0, len(ps))
}

func (ts *StorageTestSuite) TestStorage_Save_LoadLastSync() {
	s := NewPostgresStorage(ts.db)
	t := time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)

	p1 := pipe.NewPipe(1, integrations.GitHub, integrations.UsersPipe)
	p1.ServiceParams = []byte(`{"start_date":"2020-01-02"}`)
	err := s.Save(p1)
	ts.NoError(err)

	s.LoadLastSync(p1)
	ts.NotNil(p1.LastSync)
	ts.Equal(t, *p1.LastSync)
}

func (ts *StorageTestSuite) TestStorage_ClearImportFor() {
	s := NewPostgresStorage(ts.db)
	svc := pipe.NewExternalService(integrations.GitHub, 1)
	err := s.DeleteUsersFor(svc)
	ts.NoError(err)
}

func (ts *StorageTestSuite) TestStorage_DeletePipeConnections() {
	s := NewPostgresStorage(ts.db)

	p1 := pipe.NewPipe(1, integrations.GitHub, integrations.UsersPipe)
	p1.PipeStatus = pipe.NewPipeStatus(1, integrations.GitHub, integrations.UsersPipe, "test")
	svc := pipe.NewExternalService(integrations.GitHub, 1)

	err := s.DeleteIDMappings(1, svc.KeyFor(p1.ID), p1.PipeStatus.Key)
	ts.NoError(err)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestStorageTestSuite(t *testing.T) {
	suite.Run(t, new(StorageTestSuite))
}
