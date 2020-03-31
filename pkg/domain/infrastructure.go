package domain

import (
	"time"

	goauth2 "code.google.com/p/goauth2/oauth"
	"github.com/tambet/oauthplain"
)

//go:generate mockery -name TogglClient -case underscore -outpkg mocks
type TogglClient interface {
	WithAuthToken(authToken string)
	GetWorkspaceIdByToken(token string) (int, error)
	PostClients(clientsPipeID PipeID, clients interface{}) (*ClientsImport, error)
	PostProjects(projectsPipeID PipeID, projects interface{}) (*ProjectsImport, error)
	PostTasks(tasksPipeID PipeID, tasks interface{}) (*TasksImport, error)
	PostTodoLists(tasksPipeID PipeID, tasks interface{}) (*TasksImport, error)
	PostUsers(usersPipeID PipeID, users interface{}) (*UsersImport, error)
	GetTimeEntries(lastSync time.Time, userIDs, projectsIDs []int) ([]TimeEntry, error)
	AdjustRequestSize(tasks []*Task, split int) ([]*TaskRequest, error)
	Ping() error
}

//go:generate mockery -name Queue -case underscore -outpkg mocks
type Queue interface {
	ScheduleAutomaticPipesSynchronization() error
	LoadScheduledPipes() ([]*Pipe, error)
	MarkPipeSynchronized(*Pipe) error
	SchedulePipeSynchronization(workspaceID int, serviceID IntegrationID, pipeID PipeID, usersSelector UserParams) error
}

//go:generate mockery -name OAuthProvider -case underscore -outpkg mocks
type OAuthProvider interface {
	OAuth2URL(IntegrationID) string
	OAuth1Configs(IntegrationID) (*oauthplain.Config, bool)
	OAuth1Exchange(sid IntegrationID, accountName, oAuthToken, oAuthVerifier string) (*oauthplain.Token, error)
	OAuth2Exchange(sid IntegrationID, code string) (*goauth2.Token, error)
	OAuth2Configs(IntegrationID) (*goauth2.Config, bool)
	OAuth2Refresh(*goauth2.Config, *goauth2.Token) error
}
