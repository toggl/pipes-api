package cfg

import (
	"fmt"
	"strings"
	"time"
)

const (
	startStatus = "running"
)

type PipeStatus struct {
	Status        string   `json:"status,omitempty"`
	Message       string   `json:"message,omitempty"`
	SyncLog       string   `json:"sync_log,omitempty"`
	SyncDate      string   `json:"sync_date,omitempty"`
	ObjectCounts  []string `json:"object_counts,omitempty"`
	Notifications []string `json:"notifications,omitempty"`

	WorkspaceID  int    `json:"-"`
	ServiceID    string `json:"-"`
	PipeID       string `json:"-"`
	Key          string `json:"-"`
	pipesApiHost string `json:"-"`
}

func NewPipeStatus(workspaceID int, serviceID, pipeID, pipesApiHost string) *PipeStatus {
	return &PipeStatus{
		Status:       startStatus,
		SyncDate:     time.Now().Format(time.RFC3339),
		WorkspaceID:  workspaceID,
		ServiceID:    serviceID,
		PipeID:       pipeID,
		Key:          PipesKey(serviceID, pipeID),
		pipesApiHost: pipesApiHost,
	}
}

func (p *PipeStatus) AddError(err error) {
	p.Status = "error"
	p.Message = err.Error()
}

func (p *PipeStatus) Complete(objType string, notifications []string, objCount int) {
	if p.Status == "error" {
		return
	}
	p.Status = "success"
	p.Notifications = notifications
	if objCount > 0 {
		p.ObjectCounts = append(p.ObjectCounts, fmt.Sprintf("%d %s", objCount, objType))
	}
	p.SyncLog = fmt.Sprintf("%s/api/v1/integrations/%s/pipes/%s/log", p.pipesApiHost, p.ServiceID, p.PipeID)
}

func (p *PipeStatus) GenerateLog() string {
	warnings := strings.Join(p.Notifications, "\r\n")
	splitter := "------------------------------------------------"
	result := fmt.Sprintf("Log for '%s %s' (%s)\r\n%s\r\n%s.\r\n%s",
		p.ServiceID, p.PipeID, time.Now().Format(time.RFC3339), splitter, p.Message, warnings)
	return result
}
