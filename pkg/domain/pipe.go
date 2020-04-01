package domain

import (
	"fmt"
	"strings"
	"time"
)

type IntegrationID string

const (
	BaseCamp   IntegrationID = "basecamp"
	FreshBooks IntegrationID = "freshbooks"
	TogglPlan  IntegrationID = "teamweek" // should not be changed to "togglplan", because active customers lost their synchronized data.
	Asana      IntegrationID = "asana"
	GitHub     IntegrationID = "github"
)

type PipeID string

const (
	UsersPipe       PipeID = "users"
	ClientsPipe     PipeID = "clients"
	ProjectsPipe    PipeID = "projects"
	TasksPipe       PipeID = "tasks"
	TodoListsPipe   PipeID = "todolists"
	TodosPipe       PipeID = "todos"
	TimeEntriesPipe PipeID = "timeentries"
	AccountsPipe    PipeID = "accounts"
)

func NewPipe(workspaceID int, sid IntegrationID, pid PipeID) *Pipe {
	return &Pipe{
		ID:          pid,
		ServiceID:   sid,
		WorkspaceID: workspaceID,
	}
}

type Integration struct {
	ID         IntegrationID `json:"id"`
	Name       string        `json:"name"`
	Link       string        `json:"link"`
	Image      string        `json:"image"`
	AuthURL    string        `json:"auth_url,omitempty"`
	AuthType   string        `json:"auth_type,omitempty"`
	Authorized bool          `json:"authorized"`
	Pipes      []*Pipe       `json:"pipes"`
}

type Pipe struct {
	ID              PipeID        `json:"id"`
	Name            string        `json:"name"`
	Description     string        `json:"description,omitempty"`
	Automatic       bool          `json:"automatic,omitempty"`
	AutomaticOption bool          `json:"automatic_option"`
	Configured      bool          `json:"configured"`
	Premium         bool          `json:"premium"`
	ServiceParams   []byte        `json:"service_params,omitempty"`
	PipeStatus      *Status       `json:"pipe_status,omitempty"`
	WorkspaceID     int           `json:"-"`
	ServiceID       IntegrationID `json:"-"`
	UsersSelector   UserParams    `json:"-"`
	LastSync        *time.Time    `json:"-"`
	PipesApiHost    string        `json:"-"`
	WorkspaceToken  string        `json:"-"`
}

func (p *Pipe) Key() string {
	return PipesKey(p.ServiceID, p.ID)
}

func PipesKey(sid IntegrationID, pid PipeID) string {
	return fmt.Sprintf("%s:%s", sid, pid)
}

func GetSidPidFromKey(key string) (IntegrationID, PipeID) {
	ids := strings.Split(key, ":")
	return IntegrationID(ids[0]), PipeID(ids[1])
}
