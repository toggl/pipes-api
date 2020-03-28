package domain_test

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/toggl/pipes-api/pkg/domain"
	"github.com/toggl/pipes-api/pkg/domain/mocks"
)

type ServiceTestSuite struct {
	suite.Suite
	db  *sql.DB
	svc *domain.Service
}

func (ts *ServiceTestSuite) SetupTest() {
	pipeStorage := &mocks.PipesStorage{}
	importStorage := &mocks.ImportsStorage{}
	integrationStorage := &mocks.IntegrationsStorage{}
	idMappingStorage := &mocks.IDMappingsStorage{}
	togglClient := &mocks.TogglClient{}
	authorizationStorage := &mocks.AuthorizationsStorage{}
	oauthProvider := &mocks.OAuthProvider{}

	authFactory := &domain.AuthorizationFactory{
		IntegrationsStorage:   integrationStorage,
		AuthorizationsStorage: authorizationStorage,
		OAuthProvider:         oauthProvider,
	}

	pipeFactory := &domain.PipeFactory{
		AuthorizationFactory:  authFactory,
		AuthorizationsStorage: authorizationStorage,
		PipesStorage:          pipeStorage,
		ImportsStorage:        importStorage,
		IDMappingsStorage:     idMappingStorage,
		TogglClient:           togglClient,
	}

	ts.svc = &domain.Service{
		AuthorizationFactory:  authFactory,
		PipeFactory:           pipeFactory,
		PipesStorage:          pipeStorage,
		AuthorizationsStorage: authorizationStorage,
		IntegrationsStorage:   integrationStorage,
		IDMappingsStorage:     idMappingStorage,
		ImportsStorage:        importStorage,
		OAuthProvider:         oauthProvider,
		TogglClient:           togglClient,
	}
}

func (ts *ServiceTestSuite) TearDownTest() {
	ts.svc = nil
}

func (ts *ServiceTestSuite) TestService_Ready() {
	ps := &mocks.PipesStorage{}
	ps.On("IsDown").Return(false)

	tc := &mocks.TogglClient{}
	tc.On("Ping").Return(nil)

	ts.svc.PipesStorage = ps
	ts.svc.TogglClient = tc
	err := ts.svc.Ready()
	ts.Empty(err)
}

func (ts *ServiceTestSuite) TestService_Ready_IsDown() {
	ps := &mocks.PipesStorage{}
	ps.On("IsDown").Return(true)

	tc := &mocks.TogglClient{}
	tc.On("Ping").Return(nil)

	ts.svc.PipesStorage = ps
	ts.svc.TogglClient = tc
	err := ts.svc.Ready()
	ts.NotEmpty(err)
}

func (ts *ServiceTestSuite) TestService_Ready_Ping() {
	ps := &mocks.PipesStorage{}
	ps.On("IsDown").Return(false)

	tc := &mocks.TogglClient{}
	tc.On("Ping").Return(errors.New("error"))

	ts.svc.PipesStorage = ps
	ts.svc.TogglClient = tc
	err := ts.svc.Ready()
	ts.NotEmpty(err)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ServiceTestSuite))
}
