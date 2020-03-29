package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/toggl/pipes-api/pkg/integration"
)

func NewPipe(workspaceID int, sid integration.ID, pid integration.PipeID) *Pipe {
	return &Pipe{
		ID:          pid,
		ServiceID:   sid,
		WorkspaceID: workspaceID,
	}
}

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
	WorkspaceID     int                `json:"-"`
	ServiceID       integration.ID     `json:"-"`
	UsersSelector   UserParams         `json:"-"`
	LastSync        *time.Time         `json:"-"`
	PipesApiHost    string             `json:"-"`
}

func (p *Pipe) Key() string {
	return PipesKey(p.ServiceID, p.ID)
}

func PipesKey(sid integration.ID, pid integration.PipeID) string {
	return fmt.Sprintf("%s:%s", sid, pid)
}

func GetSidPidFromKey(key string) (integration.ID, integration.PipeID) {
	ids := strings.Split(key, ":")
	return integration.ID(ids[0]), integration.PipeID(ids[1])
}
