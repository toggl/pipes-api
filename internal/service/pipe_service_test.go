package service_test

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/toggl/pipes-api/internal/service"
	"github.com/toggl/pipes-api/pkg/domain"
	"github.com/toggl/pipes-api/pkg/domain/mocks"
)

type ServiceTestSuite struct {
	suite.Suite
	db  *sql.DB
	svc *service.Service
}

func (ts *ServiceTestSuite) SetupTest() {
	pipeStorage := &mocks.PipesStorage{}
	importStorage := &mocks.ImportsStorage{}
	integrationStorage := &mocks.IntegrationsStorage{}
	idMappingStorage := &mocks.IDMappingsStorage{}
	togglClient := &mocks.TogglClient{}
	authorizationStorage := &mocks.AuthorizationsStorage{}
	oauthProvider := &mocks.OAuthProvider{}

	ts.svc = &service.Service{
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

func TestNewExternalService(t *testing.T) {
	s1 := service.NewPipeIntegration(domain.BaseCamp, 1)
	s2 := service.NewPipeIntegration(domain.Asana, 2)
	s3 := service.NewPipeIntegration(domain.GitHub, 3)
	s4 := service.NewPipeIntegration(domain.FreshBooks, 4)
	s5 := service.NewPipeIntegration(domain.TeamWeek, 5)

	assert.Equal(t, domain.BaseCamp, s1.ID())
	assert.Equal(t, domain.Asana, s2.ID())
	assert.Equal(t, domain.GitHub, s3.ID())
	assert.Equal(t, domain.FreshBooks, s4.ID())
	assert.Equal(t, domain.TeamWeek, s5.ID())
}

func TestNewExternalServicePanic(t *testing.T) {
	pf := func() { service.NewPipeIntegration("Unknown", 1) }
	assert.Panics(t, pf)
}
