package storage

import (
	"os"

	"github.com/toggl/pipes-api/pkg/domain"
	"github.com/toggl/pipes-api/pkg/domain/mocks"
	"github.com/toggl/pipes-api/pkg/integration"
)

const defaultConnectionString = "dbname=pipes_test user=pipes_user host=localhost sslmode=disable port=5432"
const defaultConnStringEnv = "PIPES_API_DSN"

func getConnectionStringForTests() string {
	connString := os.Getenv(defaultConnStringEnv)
	if connString == "" {
		connString = defaultConnectionString
	}
	return connString
}

func createPipeForTests(workspaceID int, sid integration.ID, pid integration.PipeID) *domain.Pipe {
	af := &domain.AuthorizationFactory{
		IntegrationsStorage:   &mocks.IntegrationsStorage{},
		AuthorizationsStorage: &mocks.AuthorizationsStorage{},
		OAuthProvider:         &mocks.OAuthProvider{},
	}

	pf := &domain.PipeFactory{
		AuthorizationFactory:  af,
		AuthorizationsStorage: &mocks.AuthorizationsStorage{},
		PipesStorage:          &mocks.PipesStorage{},
		ImportsStorage:        &mocks.ImportsStorage{},
		IDMappingsStorage:     &mocks.IDMappingsStorage{},
		TogglClient:           &mocks.TogglClient{},
	}

	return pf.Create(workspaceID, sid, pid)
}
