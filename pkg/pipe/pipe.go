package pipe

import (
	"fmt"
	"time"

	"github.com/toggl/pipes-api/pkg/integrations"
	"github.com/toggl/pipes-api/pkg/integrations/asana"
	"github.com/toggl/pipes-api/pkg/integrations/basecamp"
	"github.com/toggl/pipes-api/pkg/integrations/freshbooks"
	"github.com/toggl/pipes-api/pkg/integrations/github"
	"github.com/toggl/pipes-api/pkg/integrations/teamweek"
	"github.com/toggl/pipes-api/pkg/toggl"
)

type QueueRunner interface {
	Queue
	Run(*Pipe)
}

type Queue interface {
	QueueAutomaticPipes() error
	GetPipesFromQueue() ([]*Pipe, error)
	SetQueuedPipeSynced(*Pipe) error
}

type Storage interface {
	Queue

	IsDown() bool
	QueuePipeAsFirst(pipe *Pipe) error

	GetAccounts(s integrations.ExternalService) (*toggl.AccountsResponse, error)
	FetchAccounts(s integrations.ExternalService) error
	ClearImportFor(s integrations.ExternalService, pid integrations.PipeID) error

	LoadPipe(workspaceID int, sid integrations.ExternalServiceID, pid integrations.PipeID) (*Pipe, error)
	LoadPipeStatus(workspaceID int, sid integrations.ExternalServiceID, pid integrations.PipeID) (*Status, error)
	LoadAuthorization(workspaceID int, sid integrations.ExternalServiceID) (*Authorization, error)
	LoadConnection(workspaceID int, key string) (*Connection, error)
	LoadReversedConnection(workspaceID int, key string) (*ReversedConnection, error)
	LoadPipes(workspaceID int) (map[string]*Pipe, error)
	LoadLastSync(p *Pipe)
	LoadPipeStatuses(workspaceID int) (map[string]*Status, error)
	LoadWorkspaceAuthorizations(workspaceID int) (map[integrations.ExternalServiceID]bool, error)

	DeletePipeByWorkspaceIDServiceID(workspaceID int, sid integrations.ExternalServiceID) error
	DeletePipeConnections(workspaceID int, pipeConnectionKey, pipeStatusKey string) (err error)

	Destroy(p *Pipe, workspaceID int) error
	DestroyAuthorization(workspaceID int, externalServiceID integrations.ExternalServiceID) error

	Save(p *Pipe) error
	SaveConnection(c *Connection) error
	SavePipeStatus(p *Status) error
	SaveAuthorization(a *Authorization) error

	GetObject(s integrations.ExternalService, pid integrations.PipeID) ([]byte, error)
	SaveObject(workspaceID int, objKey string, obj interface{}) error
}

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

type Integration struct {
	ID         integrations.ExternalServiceID `json:"id"`
	Name       string                         `json:"name"`
	Link       string                         `json:"link"`
	Image      string                         `json:"image"`
	AuthURL    string                         `json:"auth_url,omitempty"`
	AuthType   string                         `json:"auth_type,omitempty"`
	Authorized bool                           `json:"authorized"`
	Pipes      []*Pipe                        `json:"store"`
}

type Pipe struct {
	ID              integrations.PipeID `json:"id"`
	Name            string              `json:"name"`
	Description     string              `json:"description,omitempty"`
	Automatic       bool                `json:"automatic,omitempty"`
	AutomaticOption bool                `json:"automatic_option"`
	Configured      bool                `json:"configured"`
	Premium         bool                `json:"premium"`
	ServiceParams   []byte              `json:"service_params,omitempty"`
	PipeStatus      *Status             `json:"pipe_status,omitempty"`

	WorkspaceID int                            `json:"-"`
	ServiceID   integrations.ExternalServiceID `json:"-"`
	Key         string                         `json:"-"`
	Payload     []byte                         `json:"-"`
	LastSync    *time.Time                     `json:"-"`
}

func NewPipe(workspaceID int, sid integrations.ExternalServiceID, pid integrations.PipeID) *Pipe {
	return &Pipe{
		ID:          pid,
		Key:         PipesKey(sid, pid),
		ServiceID:   sid,
		WorkspaceID: workspaceID,
	}
}

func (p *Pipe) ValidatePayload(payload []byte) string {
	if p.ID == "users" && len(payload) == 0 {
		return "Missing request payload"
	}
	p.Payload = payload
	return ""
}

func PipesKey(sid integrations.ExternalServiceID, pid integrations.PipeID) string {
	return fmt.Sprintf("%s:%s", sid, pid)
}

func NewExternalService(id integrations.ExternalServiceID, workspaceID int) integrations.ExternalService {
	switch id {
	case integrations.BaseCamp:
		return &basecamp.Service{WorkspaceID: workspaceID}
	case integrations.FreshBooks:
		return &freshbooks.Service{WorkspaceID: workspaceID}
	case integrations.TeamWeek:
		return &teamweek.Service{WorkspaceID: workspaceID}
	case integrations.Asana:
		return &asana.Service{WorkspaceID: workspaceID}
	case integrations.GitHub:
		return &github.Service{WorkspaceID: workspaceID}
	default:
		panic(fmt.Sprintf("getService: Unrecognized integrations.ExternalServiceID - %s", id))
	}
}

var _ integrations.ExternalService = (*basecamp.Service)(nil)
var _ integrations.ExternalService = (*freshbooks.Service)(nil)
var _ integrations.ExternalService = (*teamweek.Service)(nil)
var _ integrations.ExternalService = (*asana.Service)(nil)
var _ integrations.ExternalService = (*github.Service)(nil)
