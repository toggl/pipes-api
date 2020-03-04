package authorization

import (
	"database/sql"
	"encoding/json"
	"flag"
	"sync"
	"testing"
	"time"

	"code.google.com/p/goauth2/oauth"
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

func (ts *StorageTestSuite) TestStorage_Set_GetAvailableAuthorizations() {
	sp := &stubOauthProvider{}
	s := NewStorage(ts.db, sp)

	res := s.GetAvailableAuthorizations("github")
	ts.Equal("", res)

	s.SetAuthorizationType("github", TypeOauth2)
	s.SetAuthorizationType("asana", TypeOauth1)

	res = s.GetAvailableAuthorizations("github")
	ts.Equal(TypeOauth2, res)

	res = s.GetAvailableAuthorizations("asana")
	ts.Equal(TypeOauth1, res)
}

func (ts *StorageTestSuite) TestStorage_Save_Load_Ok() {
	sp := &stubOauthProvider{}
	s := NewStorage(ts.db, sp)

	a := New(1, "github")

	err := s.Save(a)
	ts.NoError(err)

	aFromDb, err := s.Load(1, "github")
	ts.NoError(err)
	ts.Equal(a, aFromDb)
}

func (ts *StorageTestSuite) TestStorage_Save_Load_DbClosed() {
	cdb, err := sql.Open("postgres", getDbConnString())
	require.NoError(ts.T(), err)
	cdb.Close()

	sp := &stubOauthProvider{}
	s := NewStorage(cdb, sp)

	a := New(2, "asana")
	err = s.Save(a)
	ts.Error(err)

	con, err := s.Load(2, "asana")
	ts.Error(err)
	ts.Nil(con)
}

func (ts *StorageTestSuite) TestStorage_Save_Destroy_Ok() {
	sp := &stubOauthProvider{}
	s := NewStorage(ts.db, sp)

	a := New(1, "github")

	err := s.Save(a)
	ts.NoError(err)

	err = s.Destroy(1, "github")
	ts.NoError(err)
}

func (ts *StorageTestSuite) TestStorage_Save_LoadWorkspaceAuthorizations_Ok() {
	sp := &stubOauthProvider{}
	s := NewStorage(ts.db, sp)

	a1 := New(1, "github")
	a2 := New(1, "asana")

	err := s.Save(a1)
	ts.NoError(err)

	err = s.Save(a2)
	ts.NoError(err)

	auth, err := s.LoadWorkspaceAuthorizations(1)
	ts.NoError(err)
	ts.Equal(true, auth["github"])
	ts.Equal(true, auth["asana"])
	ts.Equal(false, auth["unknown"])
}

func (ts *StorageTestSuite) TestStorage_Refresh_Load_Ok() {
	sp := &stubOauthProvider{}
	s := NewStorage(ts.db, sp)
	s.SetAuthorizationType("github", TypeOauth2)

	a1 := New(1, "github")
	t := oauth.Token{
		AccessToken:  "123",
		RefreshToken: "456",
		Expiry:       time.Now().Add(-time.Hour),
		Extra:        nil,
	}
	b, err := json.Marshal(t)
	ts.NoError(err)
	a1.Data = b

	err = s.Refresh(a1)
	ts.NoError(err)

	aSaved, err := s.Load(1, "github")
	ts.NoError(err)
	ts.NotEqual([]byte("{}"), aSaved.Data)
}

func (ts *StorageTestSuite) TestStorage_Refresh_Oauth1() {
	sp := &stubOauthProvider{}
	s := NewStorage(ts.db, sp)
	s.SetAuthorizationType("github", TypeOauth1)

	a1 := New(1, "asana")

	err := s.Refresh(a1)
	ts.NoError(err)
}

func (ts *StorageTestSuite) TestStorage_Refresh_NotExpired() {
	sp := &stubOauthProvider{}
	s := NewStorage(ts.db, sp)
	s.SetAuthorizationType("github", TypeOauth2)

	a1 := New(1, "github")
	t := oauth.Token{
		AccessToken:  "123",
		RefreshToken: "456",
		Expiry:       time.Now().Add(time.Hour * 24),
		Extra:        nil,
	}
	b, err := json.Marshal(t)
	ts.NoError(err)
	a1.Data = b

	err = s.Refresh(a1)
	ts.NoError(err)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestStorageTestSuite(t *testing.T) {
	suite.Run(t, new(StorageTestSuite))
}
