package pipe

import (
	"fmt"
	"strings"
	"time"

	"github.com/toggl/pipes-api/pkg/integration"
	"github.com/toggl/pipes-api/pkg/integration/asana"
	"github.com/toggl/pipes-api/pkg/integration/basecamp"
	"github.com/toggl/pipes-api/pkg/integration/freshbooks"
	"github.com/toggl/pipes-api/pkg/integration/github"
	"github.com/toggl/pipes-api/pkg/integration/teamweek"
)

type Integration struct {
	ID         integration.ID `json:"id"`
	Name       string         `json:"name"`
	Link       string         `json:"link"`
	Image      string         `json:"image"`
	AuthURL    string         `json:"auth_url,omitempty"`
	AuthType   string         `json:"auth_type,omitempty"`
	Authorized bool           `json:"authorized"`
	Pipes      []*Pipe        `json:"pipes"`
}

type Pipe struct {
	ID              integration.PipeID `json:"id"`
	Name            string             `json:"name"`
	Description     string             `json:"description,omitempty"`
	Automatic       bool               `json:"automatic,omitempty"`
	AutomaticOption bool               `json:"automatic_option"`
	Configured      bool               `json:"configured"`
	Premium         bool               `json:"premium"`
	ServiceParams   []byte             `json:"service_params,omitempty"`
	PipeStatus      *Status            `json:"pipe_status,omitempty"`

	WorkspaceID   int            `json:"-"`
	ServiceID     integration.ID `json:"-"`
	Key           string         `json:"-"`
	UsersSelector []byte         `json:"-"`
	LastSync      *time.Time     `json:"-"`
}

func NewPipe(workspaceID int, sid integration.ID, pid integration.PipeID) *Pipe {
	return &Pipe{
		ID:          pid,
		Key:         PipesKey(sid, pid),
		ServiceID:   sid,
		WorkspaceID: workspaceID,
	}
}

func PipesKey(sid integration.ID, pid integration.PipeID) string {
	return fmt.Sprintf("%s:%s", sid, pid)
}

func GetSidPidFromKey(key string) (integration.ID, integration.PipeID) {
	ids := strings.Split(key, ":")
	return integration.ID(ids[0]), integration.PipeID(ids[1])
}

func NewExternalService(id integration.ID, workspaceID int) integration.Integration {
	switch id {
	case integration.BaseCamp:
		return &basecamp.Service{WorkspaceID: workspaceID}
	case integration.FreshBooks:
		return &freshbooks.Service{WorkspaceID: workspaceID}
	case integration.TeamWeek:
		return &teamweek.Service{WorkspaceID: workspaceID}
	case integration.Asana:
		return &asana.Service{WorkspaceID: workspaceID}
	case integration.GitHub:
		return &github.Service{WorkspaceID: workspaceID}
	default:
		panic(fmt.Sprintf("getService: Unrecognized integrations.ID - %s", id))
	}
}

var _ integration.Integration = (*basecamp.Service)(nil)
var _ integration.Integration = (*freshbooks.Service)(nil)
var _ integration.Integration = (*teamweek.Service)(nil)
var _ integration.Integration = (*asana.Service)(nil)
var _ integration.Integration = (*github.Service)(nil)
