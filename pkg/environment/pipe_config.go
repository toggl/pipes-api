package environment

import (
	"errors"
	"fmt"
	"time"

	"github.com/bugsnag/bugsnag-go"
)

type PipeConfig struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Description     string            `json:"description,omitempty"`
	Automatic       bool              `json:"automatic,omitempty"`
	AutomaticOption bool              `json:"automatic_option"`
	Configured      bool              `json:"configured"`
	Premium         bool              `json:"premium"`
	PipeStatus      *PipeStatusConfig `json:"pipe_status,omitempty"`
	ServiceParams   []byte            `json:"service_params,omitempty"`

	Authorization *AuthorizationConfig
	WorkspaceID   int        `json:"-"`
	ServiceID     string     `json:"-"`
	Key           string     `json:"-"`
	Payload       []byte     `json:"-"`
	LastSync      *time.Time `json:"-"`
}

func NewPipe(workspaceID int, serviceID, pipeID string) *PipeConfig {
	return &PipeConfig{
		ID:          pipeID,
		Key:         PipesKey(serviceID, pipeID),
		ServiceID:   serviceID,
		WorkspaceID: workspaceID,
	}
}

func (p *PipeConfig) ValidateServiceConfig(payload []byte) string {
	service := Create(p.ServiceID, p.WorkspaceID)
	err := service.SetParams(payload)
	if err != nil {
		return err.Error()
	}
	p.ServiceParams = payload
	return ""
}

func (p *PipeConfig) ValidatePayload(payload []byte) string {
	if p.ID == "users" && len(payload) == 0 {
		return "Missing request payload"
	}
	p.Payload = payload
	return ""
}

// BugsnagNotifyPipe notifies bugsnag with metadata for the given pipe
func (p *PipeConfig) BugsnagNotifyPipe(err error) {
	bugsnag.Notify(err, bugsnag.MetaData{
		"pipe": {
			"ID":            p.ID,
			"Name":          p.Name,
			"ServiceParams": string(p.ServiceParams),
			"WorkspaceID":   p.WorkspaceID,
			"ServiceID":     p.ServiceID,
		},
	})
	return
}

func PipesKey(serviceID, pipeID string) string {
	return fmt.Sprintf("%s:%s", serviceID, pipeID)
}

var (
	// ErrJSONParsing hides json marshalling errors from users
	ErrJSONParsing = errors.New("Failed to parse response from service, please contact support")
)
