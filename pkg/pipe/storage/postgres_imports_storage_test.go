package storage

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/toggl/pipes-api/pkg/integrations"
	"github.com/toggl/pipes-api/pkg/pipe"
)

type ImportsStorageTestSuite struct {
	suite.Suite
	db *sql.DB
}

func (ts *ImportsStorageTestSuite) SetupSuite() {
	var err error
	ts.db, err = sql.Open("postgres", getDbConnString())
	require.NoError(ts.T(), err)

	err = ts.db.Ping()
	if err != nil {
		ts.T().Skipf("Could not connect to database, db_conn_string: %v", getDbConnString())
	}
}

func (ts *ImportsStorageTestSuite) TearDownSuite() {
	ts.db.Close()
}

func (ts *ImportsStorageTestSuite) SetupTest() {
	_, err1 := ts.db.Exec(truncateAuthorizationSQL)
	_, err2 := ts.db.Exec(truncateConnectionSQL)
	_, err3 := ts.db.Exec(truncatePipesStatusSQL)
	_, err4 := ts.db.Exec(truncatePipesSQL)
	_, err5 := ts.db.Exec(truncateImportsSQL)

	ts.NoError(err1)
	ts.NoError(err2)
	ts.NoError(err3)
	ts.NoError(err4)
	ts.NoError(err5)
}

func (ts *ImportsStorageTestSuite) TestStorage_SaveObject_LoadObject() {
	s := NewPostgresImportsStorage(ts.db)

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

func (ts *ImportsStorageTestSuite) TestStorage_ClearImportFor() {
	s := NewPostgresImportsStorage(ts.db)
	svc := pipe.NewExternalService(integrations.GitHub, 1)
	err := s.DeleteUsersFor(svc)
	ts.NoError(err)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestImportsStorageTestSuite(t *testing.T) {
	suite.Run(t, new(ImportsStorageTestSuite))
}
