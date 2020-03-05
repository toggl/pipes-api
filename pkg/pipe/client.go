package pipe

import (
	"time"

	"github.com/toggl/pipes-api/pkg/integrations"
	"github.com/toggl/pipes-api/pkg/toggl"
)

//go:generate mockery -name TogglClient -case underscore -output ./mocks
type TogglClient interface {
	WithAuthToken(authToken string)
	GetWorkspaceIdByToken(token string) (int, error)
	PostClients(clientsPipeID integrations.PipeID, clients interface{}) (*toggl.ClientsImport, error)
	PostProjects(projectsPipeID integrations.PipeID, projects interface{}) (*toggl.ProjectsImport, error)
	PostTasks(tasksPipeID integrations.PipeID, tasks interface{}) (*toggl.TasksImport, error)
	PostTodoLists(tasksPipeID integrations.PipeID, tasks interface{}) (*toggl.TasksImport, error)
	PostUsers(usersPipeID integrations.PipeID, users interface{}) (*toggl.UsersImport, error)
	GetTimeEntries(lastSync time.Time, userIDs, projectsIDs []int) ([]toggl.TimeEntry, error)
	AdjustRequestSize(tasks []*toggl.Task, split int) ([]*toggl.TaskRequest, error)
	Ping() error
}
