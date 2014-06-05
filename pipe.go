package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/toggl/bugsnag"
	"strings"
)

type Pipe struct {
	ID              string      `json:"id"`
	Name            string      `json:"name"`
	Description     string      `json:"description,omitempty"`
	AccountName     string      `json:"account_name,omitempty"`
	Automatic       bool        `json:"automatic,omitempty"`
	AutomaticOption bool        `json:"automatic_option"`
	Configured      bool        `json:"configured"`
	Premium         bool        `json:"premium"`
	PipeStatus      *PipeStatus `json:"pipe_status,omitempty"`
	ServiceParams   []byte      `json:"service_params,omitempty"`

	authorization *Authorization
	workspaceID   int
	serviceID     string
	pipeID        string
	key           string
	payload       []byte
}

const (
	selectPipesSQL = `SELECT workspace_id, key, data
    FROM pipes WHERE workspace_id = $1
  `
	selectAutomaticPipesSQL = `SELECT workspace_id, key, data
		FROM pipes WHERE
		data->>'automatic' = 'true'
  `
	singlePipesSQL = `SELECT workspace_id, key, data
    FROM pipes WHERE workspace_id = $1
    AND key = $2 LIMIT 1
  `
	deletePipeSQL = `DELETE FROM pipes
    WHERE workspace_id = $1
    AND key = $2
  `
	insertPipesSQL = `
    WITH existing_pipe AS (
      UPDATE pipes SET data = $3
      WHERE workspace_id = $1 AND key = $2
      RETURNING key
    ),
    inserted_pipe AS (
      INSERT INTO pipes(workspace_id, key, data)
      SELECT $1, $2, $3
      WHERE NOT EXISTS (SELECT 1 FROM existing_pipe)
      RETURNING key
    )
    SELECT * FROM inserted_pipe
    UNION
    SELECT * FROM existing_pipe
  `
)

func NewPipe(workspaceID int, serviceID, pipeID string) *Pipe {
	return &Pipe{
		ID:          pipeID,
		Name:        strings.Title(pipeID),
		key:         pipesKey(serviceID, pipeID),
		serviceID:   serviceID,
		workspaceID: workspaceID,
	}
}

func pipesKey(serviceID, pipeID string) string {
	return fmt.Sprintf("%s:%s", serviceID, pipeID)
}

func (p *Pipe) save() error {
	p.Configured = true
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = db.Exec(insertPipesSQL, p.workspaceID, p.key, b)
	if err != nil {
		return err
	}
	return nil
}

func (p *Pipe) validateServiceConfig(payload []byte) string {
	service := getService(p.serviceID, p.workspaceID)
	err := service.setParams(payload)
	if err != nil {
		return err.Error()
	}
	p.ServiceParams = payload
	return ""
}

func (p *Pipe) validatePayload(payload []byte) string {
	if p.ID == "users" && len(payload) == 0 {
		return "Missing request payload"
	}
	p.payload = payload
	return ""
}

func (p *Pipe) load(rows *sql.Rows) error {
	var wid int
	var b []byte
	var key string
	if err := rows.Scan(&wid, &key, &b); err != nil {
		return err
	}
	err := json.Unmarshal(b, p)
	if err != nil {
		return err
	}
	p.key = key
	p.workspaceID = wid
	p.serviceID = strings.Split(key, ":")[0]
	return nil
}

func (p *Pipe) NewStatus() error {
	p.PipeStatus = NewPipeStatus(p.workspaceID, p.serviceID, p.ID)
	return p.PipeStatus.save()
}

func (p *Pipe) Service() Service {
	service := getService(p.serviceID, p.workspaceID)
	service.setParams(p.ServiceParams)
	loadAuth(service)
	return service
}

func (p *Pipe) loadAuth() error {
	service := getService(p.serviceID, p.workspaceID)
	auth, err := loadAuth(service)
	if err != nil {
		return err
	}
	p.authorization = auth
	return nil
}

func (p *Pipe) run() {
	var err error
	defer func() { p.endSync(true, err) }()

	if err = p.NewStatus(); err != nil {
		return
	}
	if err = p.loadAuth(); err != nil {
		return
	}
	if err = p.fetchObjects(false); err != nil {
		return
	}
	if err = p.postObjects(false); err != nil {
		return
	}
}

func (p *Pipe) fetchObjects(saveStatus bool) (err error) {
	switch p.ID {
	case "users":
		err = fetchUsers(p)
	case "projects":
		err = fetchProjects(p)
	case "todolists":
		err = fetchTodoLists(p)
	case "todos", "tasks":
		err = fetchTasks(p)
	default:
		panic(fmt.Sprintf("fetchObjects: Unrecognized pipeID - %s", p.ID))
	}
	return p.endSync(saveStatus, err)
}

func (p *Pipe) postObjects(saveStatus bool) (err error) {
	switch p.ID {
	case "users":
		err = postUsers(p)
	case "projects":
		err = postProjects(p)
	case "todolists":
		err = postTodoLists(p)
	case "todos":
		err = postTasks(p)
	default:
		panic(fmt.Sprintf("postObjects: Unrecognized pipeID - %s", p.ID))
	}
	return p.endSync(saveStatus, err)
}

func (p *Pipe) endSync(saveStatus bool, err error) error {
	if err != nil && saveStatus {
		p.PipeStatus.addError(err)
	}
	if saveStatus {
		if err := p.PipeStatus.save(); err != nil {
			bugsnag.Notify(err)
			return err
		}
	}
	return err
}

func (p *Pipe) destroy(workspaceID int) error {
	_, err := db.Exec(deletePipeSQL, workspaceID, p.key)
	return err
}

func loadPipe(workspaceID int, serviceID, pipeID string) (*Pipe, error) {
	key := pipesKey(serviceID, pipeID)
	rows, err := db.Query(singlePipesSQL, workspaceID, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, rows.Err()
	}
	var pipe Pipe
	if err := pipe.load(rows); err != nil {
		return nil, err
	}
	return &pipe, nil
}

func loadPipes(workspaceID int) (map[string]*Pipe, error) {
	pipes := make(map[string]*Pipe)
	rows, err := db.Query(selectPipesSQL, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var pipe Pipe
		if err := pipe.load(rows); err != nil {
			return nil, err
		}
		pipes[pipe.key] = &pipe
	}
	return pipes, nil
}

func loadAutomaticPipes() ([]*Pipe, error) {
	var pipes []*Pipe
	rows, err := db.Query(selectAutomaticPipesSQL)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var pipe Pipe
		if err := pipe.load(rows); err != nil {
			return nil, err
		}
		pipes = append(pipes, &pipe)
	}
	return pipes, nil
}
