package storage

import (
	"os"

	"github.com/toggl/pipes-api/pkg/domain"
)

const defaultConnectionString = "dbname=pipes_test user=pipes_user host=127.0.0.1 sslmode=disable port=54320"
const defaultConnStringEnv = "PIPES_API_POSTGRES_DSN"

func getConnectionStringForTests() string {
	connString := os.Getenv(defaultConnStringEnv)
	if connString == "" {
		connString = defaultConnectionString
	}
	return connString
}

func createPipeForTests(workspaceID int, sid domain.IntegrationID, pid domain.PipeID) *domain.Pipe {
	return domain.NewPipe(workspaceID, sid, pid)
}
