package pipe

import (
	"errors"
	"fmt"
	"time"

	"github.com/toggl/pipes-api/pkg/integrations"
	"github.com/toggl/pipes-api/pkg/integrations/asana"
	"github.com/toggl/pipes-api/pkg/integrations/basecamp"
	"github.com/toggl/pipes-api/pkg/integrations/freshbooks"
	"github.com/toggl/pipes-api/pkg/integrations/github"
	"github.com/toggl/pipes-api/pkg/integrations/teamweek"
)

// ErrJSONParsing hides json marshalling errors from users
var ErrJSONParsing = errors.New("failed to parse response from service, please contact support")

type Integration struct {
	ID         integrations.ExternalServiceID `json:"id"`
	Name       string                         `json:"name"`
	Link       string                         `json:"link"`
	Image      string                         `json:"image"`
	AuthURL    string                         `json:"auth_url,omitempty"`
	AuthType   string                         `json:"auth_type,omitempty"`
	Authorized bool                           `json:"authorized"`
	Pipes      []*Pipe                        `json:"pipes"`
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
