package storage

import (
	"os"

	"github.com/toggl/pipes-api/pkg/domain"
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

func createPipeForTests(workspaceID int, sid domain.ID, pid domain.PipeID) *domain.Pipe {
	return domain.NewPipe(workspaceID, sid, pid)
}
