package domain_test

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/toggl/pipes-api/internal/storage"
	"github.com/toggl/pipes-api/pkg/domain"
	"github.com/toggl/pipes-api/pkg/domain/mocks"
	"github.com/toggl/pipes-api/pkg/integration"
	"github.com/toggl/pipes-api/pkg/toggl/client"
)

type ServiceTestSuite struct {
	suite.Suite
	db *sql.DB
}

func (ts *ServiceTestSuite) TestService_Set_GetAvailableAuthorizations() {
	s := storage.NewPipeStorage(ts.db)
	as := storage.NewAuthorizationStorage(ts.db)
	ims := storage.NewImportStorage(ts.db)
	idms := storage.NewIdMappingStorageStorage(ts.db)
	api := client.NewTogglApiClient("https://localhost")
	op := &mocks.OAuthProvider{}
	q := &mocks.Queue{}

	is := &mocks.IntegrationsStorage{}
	is.On("SaveAuthorizationType", mock.Anything, mock.Anything).Return(nil)
	is.On("LoadAuthorizationType", integration.GitHub).Return(domain.TypeOauth2, nil)
	is.On("LoadAuthorizationType", integration.Asana).Return(domain.TypeOauth1, nil)

	af := &domain.AuthorizationFactory{
		IntegrationsStorage:   is,
		AuthorizationsStorage: as,
		OAuthProvider:         op,
	}

	pf := &domain.PipeFactory{
		AuthorizationFactory:  af,
		AuthorizationsStorage: as,
		PipesStorage:          s,
		ImportsStorage:        ims,
		IDMappingsStorage:     idms,
		TogglClient:           api,
	}

	svc := &domain.Service{
		AuthorizationFactory:  af,
		PipeFactory:           pf,
		PipesStorage:          s,
		AuthorizationsStorage: as,
		IntegrationsStorage:   is,
		IDMappingsStorage:     idms,
		ImportsStorage:        ims,
		OAuthProvider:         op,
		TogglClient:           api,
		Queue:                 q,
	}

	res, err := svc.IntegrationsStorage.LoadAuthorizationType(integration.GitHub)
	ts.NoError(err)
	ts.Equal(domain.TypeOauth2, res)

	err = svc.IntegrationsStorage.SaveAuthorizationType(integration.GitHub, domain.TypeOauth2)
	ts.NoError(err)
	err = svc.IntegrationsStorage.SaveAuthorizationType(integration.Asana, domain.TypeOauth1)
	ts.NoError(err)

	res, err = svc.IntegrationsStorage.LoadAuthorizationType(integration.GitHub)
	ts.NoError(err)
	ts.Equal(domain.TypeOauth2, res)

	res, err = svc.IntegrationsStorage.LoadAuthorizationType(integration.Asana)
	ts.NoError(err)
	ts.Equal(domain.TypeOauth1, res)
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestServiceTestSuite(t *testing.T) {
	suite.Run(t, new(ServiceTestSuite))
}
