package pipe

import (
	"fmt"
	"strings"
	"time"

	"github.com/toggl/pipes-api/pkg/integrations"
)

const (
	StatusRunning = "running"
	StatusError   = "error"
	StatusSuccess = "success"
)

type Status struct {
	Status        string   `json:"status,omitempty"`
	Message       string   `json:"message,omitempty"`
	SyncLog       string   `json:"sync_log,omitempty"`
	SyncDate      string   `json:"sync_date,omitempty"`
	ObjectCounts  []string `json:"object_counts,omitempty"`
	Notifications []string `json:"notifications,omitempty"`

	WorkspaceID  int                            `json:"-"`
	ServiceID    integrations.ExternalServiceID `json:"-"`
	PipeID       integrations.PipeID            `json:"-"`
	Key          string                         `json:"-"`
	PipesApiHost string                         `json:"-"`
}

func NewPipeStatus(workspaceID int, externalServiceID integrations.ExternalServiceID, pipeID integrations.PipeID, pipesApiHost string) *Status {
	return &Status{
		Status:       StatusRunning,
		SyncDate:     time.Now().Format(time.RFC3339),
		WorkspaceID:  workspaceID,
		ServiceID:    externalServiceID,
		PipeID:       pipeID,
		Key:          PipesKey(externalServiceID, pipeID),
		PipesApiHost: pipesApiHost,
	}
}

func (p *Status) AddError(err error) {
	p.Status = StatusError
	p.Message = err.Error()
}

func (p *Status) Complete(objType integrations.PipeID, notifications []string, objCount int) {
	if p.Status == StatusError {
		return
	}
	p.Status = StatusSuccess
	p.Notifications = notifications
	if objCount > 0 {
		p.ObjectCounts = append(p.ObjectCounts, fmt.Sprintf("%d %s", objCount, objType))
	}
	p.SyncLog = fmt.Sprintf("%s/api/v1/integrations/%s/pipes/%s/log", p.PipesApiHost, p.ServiceID, p.PipeID)
}

func (p *Status) GenerateLog() string {
	warnings := strings.Join(p.Notifications, "\r\n")
	splitter := "------------------------------------------------"
	result := fmt.Sprintf("Log for '%s %s' (%s)\r\n%s\r\n%s.\r\n%s",
		p.ServiceID, p.PipeID, time.Now().Format(time.RFC3339), splitter, p.Message, warnings)
	return result
}
