package service_test

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/toggl/pipes-api/internal/service"
	"github.com/toggl/pipes-api/pkg/domain/mocks"
)

type ServiceTestSuite struct {
	suite.Suite
	db                   *sql.DB
	svc                  *service.PipeService
	pipeStorage          *mocks.PipesStorage
	importStorage        *mocks.ImportsStorage
	integrationStorage   *mocks.IntegrationsStorage
	idMappingStorage     *mocks.IDMappingsStorage
	authorizationStorage *mocks.AuthorizationsStorage
	togglClient          *mocks.TogglClient
	oauthProvider        *mocks.OAuthProvider
}

func (ts *ServiceTestSuite) SetupTest() {
	ts.pipeStorage = &mocks.PipesStorage{}
	ts.importStorage = &mocks.ImportsStorage{}
	ts.integrationStorage = &mocks.IntegrationsStorage{}
	ts.idMappingStorage = &mocks.IDMappingsStorage{}
	ts.togglClient = &mocks.TogglClient{}
	ts.authorizationStorage = &mocks.AuthorizationsStorage{}
	ts.oauthProvider = &mocks.OAuthProvider{}

	ts.svc = service.NewPipeService(
		ts.pipeStorage,
		ts.authorizationStorage,
		ts.integrationStorage,
		ts.idMappingStorage,
		ts.importStorage,
		ts.oauthProvider,
		ts.togglClient,
	)
}

func (ts *ServiceTestSuite) TearDownTest() {
	ts.svc = nil
	ts.pipeStorage = nil
	ts.importStorage = nil
	ts.integrationStorage = nil
	ts.idMappingStorage = nil
	ts.authorizationStorage = nil
	ts.togglClient = nil
	ts.oauthProvider = nil
}

func (ts *ServiceTestSuite) TestService_Ready() {
	ts.pipeStorage.On("IsDown").Return(false)
	ts.togglClient.On("Ping").Return(nil)
	err := ts.svc.Ready()
	ts.Empty(err)
}

func (ts *ServiceTestSuite) TestService_Ready_IsDown() {
	ts.pipeStorage.On("IsDown").Return(true)
	ts.togglClient.On("Ping").Return(nil)

	err := ts.svc.Ready()
	ts.NotEmpty(err)
}

func (ts *ServiceTestSuite) TestService_Ready_Ping() {
	ts.pipeStorage.On("IsDown").Return(false)
	ts.togglClient.On("Ping").Return(errors.New("error"))

	err := ts.svc.Ready()
	ts.NotEmpty(err)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ServiceTestSuite))
}
