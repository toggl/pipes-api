package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type PipeStatus struct {
	Status        string   `json:"status,omitempty"`
	Message       string   `json:"message,omitempty"`
	SyncLog       string   `json:"sync_log,omitempty"`
	SyncDate      string   `json:"sync_date,omitempty"`
	ObjectCounts  []string `json:"object_counts,omitempty"`
	Notifications []string `json:"notifications,omitempty"`

	workspaceID int
	serviceID   string
	pipeID      string
	key         string
}

const (
	startStatus         = "running"
	selectPipeStatusSQL = `SELECT key, data
    FROM pipes_status WHERE workspace_id = $1`
	singlePipeStatusSQL = `SELECT data
    FROM pipes_status WHERE workspace_id = $1 AND key = $2 LIMIT 1`
	insertPipeStatusSQL = `
    WITH existing_status AS (
      UPDATE pipes_status SET data = $3
      WHERE workspace_id = $1 AND key = $2
      RETURNING key
    ),
    inserted_status AS (
      INSERT INTO pipes_status(workspace_id, key, data)
      SELECT $1, $2, $3
      WHERE NOT EXISTS (SELECT 1 FROM existing_status)
      RETURNING key
    )
    SELECT * FROM inserted_status
    UNION
    SELECT * FROM existing_status
    `
)

func NewPipeStatus(workspaceID int, serviceID, pipeID string) *PipeStatus {
	return &PipeStatus{
		Status:      startStatus,
		SyncDate:    time.Now().Format(time.RFC3339),
		workspaceID: workspaceID,
		serviceID:   serviceID,
		pipeID:      pipeID,
		key:         pipesKey(serviceID, pipeID),
	}
}

func (p *PipeStatus) save() error {
	if p.Status == "success" && len(p.ObjectCounts) > 0 {
		p.Message = fmt.Sprintf("%s successfully imported", strings.Join(p.ObjectCounts, ", "))
	}
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = db.Exec(insertPipeStatusSQL, p.workspaceID, p.key, b)
	if err != nil {
		return err
	}
	return nil
}

func (p *PipeStatus) load(rows *sql.Rows) error {
	var b []byte
	var key string
	if err := rows.Scan(&key, &b); err != nil {
		return err
	}
	err := json.Unmarshal(b, p)
	if err != nil {
		return err
	}
	p.key = key
	return nil
}

func (p *PipeStatus) addError(err error) {
	p.Status = "error"
	p.Message = err.Error()
}

func (p *PipeStatus) complete(objType string, notifications []string, objCount int) {
	p.Status = "success"
	p.Notifications = notifications
	p.ObjectCounts = append(p.ObjectCounts, fmt.Sprintf("%d %s", objCount, objType))
	p.SyncLog = fmt.Sprintf("%s/api/v1/integrations/%s/pipes/%s/log",
		urls.PipesAPIHost[*environment], p.serviceID, p.pipeID)
}

func (p *PipeStatus) generateLog() string {
	warnings := strings.Join(p.Notifications, "\r\n")
	splitter := "------------------------------------------------"
	result := fmt.Sprintf("Import log for '%s %s' (%s)\r\n%s\r\n%s.\r\n%s",
		p.serviceID, p.pipeID, time.Now().Format(time.RFC3339), splitter, p.Message, warnings)
	return result
}

func loadPipeStatus(workspaceID int, serviceID, pipeID string) (*PipeStatus, error) {
	key := pipesKey(serviceID, pipeID)
	rows, err := db.Query(singlePipeStatusSQL, workspaceID, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, rows.Err()
	}
	var b []byte
	if err := rows.Scan(&b); err != nil {
		return nil, err
	}
	var pipeStatus PipeStatus
	if err = json.Unmarshal(b, &pipeStatus); err != nil {
		return nil, err
	}
	pipeStatus.workspaceID = workspaceID
	pipeStatus.serviceID = serviceID
	pipeStatus.pipeID = pipeID
	return &pipeStatus, nil
}

func loadPipeStatuses(workspaceID int) (map[string]*PipeStatus, error) {
	pipeStatuses := make(map[string]*PipeStatus)
	rows, err := db.Query(selectPipeStatusSQL, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var pipeStatus PipeStatus
		if err := pipeStatus.load(rows); err != nil {
			return nil, err
		}
		pipeStatuses[pipeStatus.key] = &pipeStatus
	}
	return pipeStatuses, nil
}
