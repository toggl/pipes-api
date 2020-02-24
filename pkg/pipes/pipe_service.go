package pipes

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/bugsnag/bugsnag-go"

	"github.com/toggl/pipes-api/pkg/cfg"
	"github.com/toggl/pipes-api/pkg/integrations"
	"github.com/toggl/pipes-api/pkg/storage"
	"github.com/toggl/pipes-api/pkg/toggl"
)

type PipeService struct {
	Storage               *storage.Storage
	AuthorizationService  *AuthorizationService
	PipesApiHost          string
	OAuthService          *cfg.OAuthService
	AvailableIntegrations []*cfg.Integration
	TogglService          *toggl.Service
	ConnectionService     *ConnectionService
}

func (ps *PipeService) LoadPipe(workspaceID int, serviceID, pipeID string) (*cfg.Pipe, error) {
	key := cfg.PipesKey(serviceID, pipeID)
	return ps.LoadPipeWithKey(workspaceID, key)
}

func (ps *PipeService) LoadPipeWithKey(workspaceID int, key string) (*cfg.Pipe, error) {
	rows, err := ps.Storage.Query(singlePipesSQL, workspaceID, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, rows.Err()
	}
	var pipe cfg.Pipe
	if err := ps.Load(rows, &pipe); err != nil {
		return nil, err
	}
	return &pipe, nil
}

func (ps *PipeService) LoadPipes(workspaceID int) (map[string]*cfg.Pipe, error) {
	pipes := make(map[string]*cfg.Pipe)
	rows, err := ps.Storage.Query(selectPipesSQL, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var pipe cfg.Pipe
		if err := ps.Load(rows, &pipe); err != nil {
			return nil, err
		}
		pipes[pipe.Key] = &pipe
	}
	return pipes, nil
}

func (ps *PipeService) GetPipesFromQueue() ([]*cfg.Pipe, error) {
	var pipes []*cfg.Pipe
	rows, err := ps.Storage.Query(selectPipesFromQueueSQL)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var workspaceID int
		var key string
		err := rows.Scan(&workspaceID, &key)
		if err != nil {
			return nil, err
		}

		if workspaceID > 0 && len(key) > 0 {
			pipe, err := ps.LoadPipeWithKey(workspaceID, key)
			if err != nil {
				return nil, err
			}
			pipes = append(pipes, pipe)
		}
	}
	return pipes, nil
}

func (ps *PipeService) SetQueuedPipeSynced(pipe *cfg.Pipe) error {
	_, err := ps.Storage.Exec(setQueuedPipeSyncedSQL, pipe.WorkspaceID, pipe.Key)
	return err
}

func (ps *PipeService) Save(p *cfg.Pipe) error {
	p.Configured = true
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = ps.Storage.Exec(insertPipesSQL, p.WorkspaceID, p.Key, b)
	if err != nil {
		return err
	}
	return nil
}

func (ps *PipeService) Load(rows *sql.Rows, p *cfg.Pipe) error {
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
	p.Key = key
	p.WorkspaceID = wid
	p.ServiceID = strings.Split(key, ":")[0]
	return nil
}

func (ps *PipeService) ServiceFor(p *cfg.Pipe) (integrations.Service, error) {
	service := integrations.GetService(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return service, err
	}
	if _, err := ps.AuthorizationService.LoadAuth(service); err != nil {
		return service, err
	}
	return service, nil
}

func (ps *PipeService) loadAuthFor(p *cfg.Pipe) error {
	service := integrations.GetService(p.ServiceID, p.WorkspaceID)
	auth, err := ps.AuthorizationService.LoadAuth(service)
	if err != nil {
		return err
	}
	if err = ps.AuthorizationService.Refresh(auth); err != nil {
		return err
	}
	p.Authorization = auth
	return nil
}

func (ps *PipeService) NewStatus(p *cfg.Pipe) error {
	ps.loadLastSync(p)
	p.PipeStatus = cfg.NewPipeStatus(p.WorkspaceID, p.ServiceID, p.ID, ps.PipesApiHost)
	return ps.savePipeStatus(p.PipeStatus)
}

func (ps *PipeService) Run(p *cfg.Pipe) {
	var err error
	defer func() {
		err := ps.endSync(p, true, err)
		log.Println(err)
	}()

	if err = ps.NewStatus(p); err != nil {
		p.BugsnagNotifyPipe(err)
		return
	}
	if err = ps.loadAuthFor(p); err != nil {
		p.BugsnagNotifyPipe(err)
		return
	}
	if err = ps.FetchObjects(p, false); err != nil {
		p.BugsnagNotifyPipe(err)
		return
	}
	if err = ps.postObjects(p, false); err != nil {
		p.BugsnagNotifyPipe(err)
		return
	}
}

func (ps *PipeService) loadLastSync(p *cfg.Pipe) {
	err := ps.Storage.QueryRow(lastSyncSQL, p.WorkspaceID, p.Key).Scan(&p.LastSync)
	if err != nil {
		var err error
		t := time.Now()
		date := struct {
			StartDate string `json:"start_date"`
		}{}
		if err = json.Unmarshal(p.ServiceParams, &date); err == nil {
			t, _ = time.Parse("2006-01-02", date.StartDate)
		}
		p.LastSync = &t
	}
}

func (ps *PipeService) FetchObjects(p *cfg.Pipe, saveStatus bool) (err error) {
	switch p.ID {
	case "users":
		err = ps.fetchUsers(p)
	case "projects":
		err = ps.fetchProjects(p)
	case "todolists":
		err = ps.fetchTodoLists(p)
	case "todos", "tasks":
		err = ps.fetchTasks(p)
	case "timeentries":
		err = ps.fetchTimeEntries(p)
	default:
		panic(fmt.Sprintf("FetchObjects: Unrecognized pipeID - %s", p.ID))
	}
	return ps.endSync(p, saveStatus, err)
}

func (ps *PipeService) postObjects(p *cfg.Pipe, saveStatus bool) (err error) {
	switch p.ID {
	case "users":
		err = ps.postUsers(p)
	case "projects":
		err = ps.postProjects(p)
	case "todolists":
		err = ps.postTodoLists(p)
	case "todos", "tasks":
		err = ps.postTasks(p)
		err = ps.postTasks(p)
	case "timeentries":
		var service integrations.Service
		service, err = ps.ServiceFor(p)
		if err != nil {
			break
		}
		err = ps.postTimeEntries(p, service)
	default:
		panic(fmt.Sprintf("postObjects: Unrecognized pipeID - %s", p.ID))
	}
	return ps.endSync(p, saveStatus, err)
}

func (ps *PipeService) Destroy(p *cfg.Pipe, workspaceID int) error {
	tx, err := ps.Storage.Begin()
	if err != nil {
		return err
	}
	if _, err = tx.Exec(deletePipeSQL, workspaceID, p.Key); err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			return rollbackErr
		}
		return err
	}
	if _, err = tx.Exec(deletePipeStatusSQL, workspaceID, p.Key); err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			return rollbackErr
		}
		return err
	}
	return tx.Commit()
}

func (ps *PipeService) ClearPipeConnections(p *cfg.Pipe) (err error) {
	s, err := ps.ServiceFor(p)
	if err != nil {
		return
	}

	pipeStatus, err := ps.LoadPipeStatus(p.WorkspaceID, p.ServiceID, p.ID)
	if err != nil {
		return
	}

	key := s.KeyFor(p.ID)

	tx, err := ps.Storage.Begin()
	if err != nil {
		return
	}
	_, err = tx.Exec(deletePipeConnectionsSQL, p.WorkspaceID, key)
	if err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			err = rollbackErr
		}

		return
	}
	_, err = tx.Exec(deletePipeStatusSQL, p.WorkspaceID, pipeStatus.Key)
	if err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			err = rollbackErr
		}

	}

	return
}

func (ps *PipeService) LoadPipeStatus(workspaceID int, serviceID, pipeID string) (*cfg.PipeStatus, error) {
	key := cfg.PipesKey(serviceID, pipeID)
	rows, err := ps.Storage.Query(singlePipeStatusSQL, workspaceID, key)
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
	var pipeStatus cfg.PipeStatus
	if err = json.Unmarshal(b, &pipeStatus); err != nil {
		return nil, err
	}
	pipeStatus.WorkspaceID = workspaceID
	pipeStatus.ServiceID = serviceID
	pipeStatus.PipeID = pipeID
	return &pipeStatus, nil
}

func (ps *PipeService) loadPipeStatuses(workspaceID int) (map[string]*cfg.PipeStatus, error) {
	pipeStatuses := make(map[string]*cfg.PipeStatus)
	rows, err := ps.Storage.Query(selectPipeStatusSQL, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var pipeStatus cfg.PipeStatus
		if err := ps.loadPipeStatusFromStore(rows, &pipeStatus); err != nil {
			return nil, err
		}
		pipeStatuses[pipeStatus.Key] = &pipeStatus
	}
	return pipeStatuses, nil
}

func (ps *PipeService) savePipeStatus(p *cfg.PipeStatus) error {
	if p.Status == "success" {
		if len(p.ObjectCounts) > 0 {
			p.Message = fmt.Sprintf("%s successfully imported/exported", strings.Join(p.ObjectCounts, ", "))
		} else {
			p.Message = fmt.Sprintf("No new %s were imported/exported", p.PipeID)
		}
	}
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = ps.Storage.Exec(insertPipeStatusSQL, p.WorkspaceID, p.Key, b)
	if err != nil {
		return err
	}
	return nil
}

func (ps *PipeService) loadPipeStatusFromStore(rows *sql.Rows, p *cfg.PipeStatus) error {
	var b []byte
	var key string
	if err := rows.Scan(&key, &b); err != nil {
		return err
	}
	err := json.Unmarshal(b, p)
	if err != nil {
		return err
	}
	p.Key = key
	return nil
}

func (ps *PipeService) endSync(p *cfg.Pipe, saveStatus bool, err error) error {
	if !saveStatus {
		return err
	}

	if err != nil {
		// If it is JSON marshalling error suppress it for status
		if _, ok := err.(*json.UnmarshalTypeError); ok {
			err = cfg.ErrJSONParsing
		}
		p.PipeStatus.AddError(err)
	}
	if err = ps.savePipeStatus(p.PipeStatus); err != nil {
		p.BugsnagNotifyPipe(err)
		return err
	}

	return nil
}

func (ps *PipeService) WorkspaceIntegrations(workspaceID int) ([]cfg.Integration, error) {
	authorizations, err := ps.AuthorizationService.LoadAuthorizations(workspaceID)
	if err != nil {
		return nil, err
	}
	workspacePipes, err := ps.LoadPipes(workspaceID)
	if err != nil {
		return nil, err
	}
	pipeStatuses, err := ps.loadPipeStatuses(workspaceID)
	if err != nil {
		return nil, err
	}

	var integrations []cfg.Integration
	for j := range ps.AvailableIntegrations {
		var integration = *ps.AvailableIntegrations[j]
		integration.AuthURL = ps.OAuthService.OAuth2URL(integration.ID)
		integration.Authorized = authorizations[integration.ID]
		var pipes []*cfg.Pipe
		for i := range integration.Pipes {
			var pipe = *integration.Pipes[i]
			key := cfg.PipesKey(integration.ID, pipe.ID)
			existingPipe := workspacePipes[key]
			if existingPipe != nil {
				pipe.Automatic = existingPipe.Automatic
				pipe.Configured = existingPipe.Configured
			}

			pipe.PipeStatus = pipeStatuses[key]
			pipes = append(pipes, &pipe)
		}
		integration.Pipes = pipes
		integrations = append(integrations, integration)
	}
	return integrations, nil
}

func (ps *PipeService) fetchTimeEntries(p *cfg.Pipe) error {
	return nil
}

func (ps *PipeService) postTimeEntries(p *cfg.Pipe, service integrations.Service) error {
	var err error
	var entriesCon *Connection
	var usersCon, tasksCon, projectsCon *ReversedConnection
	if usersCon, err = ps.ConnectionService.LoadConnectionRev(service, "users"); err != nil {
		return err
	}
	if tasksCon, err = ps.ConnectionService.LoadConnectionRev(service, "tasks"); err != nil {
		return err
	}
	if projectsCon, err = ps.ConnectionService.LoadConnectionRev(service, "projects"); err != nil {
		return err
	}
	if entriesCon, err = ps.ConnectionService.LoadConnection(service, "time_entries"); err != nil {
		return err
	}

	if p.LastSync == nil {
		currentTime := time.Now()
		p.LastSync = &currentTime
	}

	timeEntries, err := ps.TogglService.GetTimeEntries(
		p.Authorization.WorkspaceToken, *p.LastSync,
		usersCon.getKeys(), projectsCon.getKeys(),
	)
	if err != nil {
		return err
	}

	for _, entry := range timeEntries {
		entry.ForeignID = strconv.Itoa(entriesCon.Data[strconv.Itoa(entry.ID)])
		entry.ForeignTaskID = strconv.Itoa(tasksCon.getInt(entry.TaskID))
		entry.ForeignUserID = strconv.Itoa(usersCon.getInt(entry.UserID))
		entry.ForeignProjectID = strconv.Itoa(projectsCon.getInt(entry.ProjectID))

		entryID, err := service.ExportTimeEntry(&entry)
		if err != nil {
			bugsnag.Notify(err, bugsnag.MetaData{
				"Workspace": {
					"ID": service.GetWorkspaceID(),
				},
				"Entry": {
					"ID":        entry.ID,
					"TaskID":    entry.TaskID,
					"UserID":    entry.UserID,
					"ProjectID": entry.ProjectID,
				},
				"Foreign Entry": {
					"ForeignID":        entry.ForeignID,
					"ForeignTaskID":    entry.ForeignTaskID,
					"ForeignUserID":    entry.ForeignUserID,
					"ForeignProjectID": entry.ForeignProjectID,
				},
			})
			p.PipeStatus.AddError(err)
		} else {
			entriesCon.Data[strconv.Itoa(entry.ID)] = entryID
		}
	}

	if err := ps.ConnectionService.Save(entriesCon); err != nil {
		return err
	}
	p.PipeStatus.Complete("timeentries", []string{}, len(timeEntries))
	return nil
}

func (ps *PipeService) QueueAutomaticPipes() error {
	_, err := ps.Storage.Exec(queueAutomaticPipesSQL)
	return err
}

func (ps *PipeService) DeletePipeByWorkspaceIDServiceID(workspaceID int, serviceID string) error {
	_, err := ps.Storage.Exec(deletePipeSQL, workspaceID, serviceID+"%")
	return err
}

func (ps *PipeService) QueuePipeAsFirst(pipe *cfg.Pipe) error {
	_, err := ps.Storage.Exec(queuePipeAsFirstSQL, pipe.WorkspaceID, pipe.Key)
	return err
}

const (
	selectPipesSQL = `SELECT workspace_id, Key, data
    FROM pipes WHERE workspace_id = $1
  `
	singlePipesSQL = `SELECT workspace_id, Key, data
    FROM pipes WHERE workspace_id = $1
    AND Key = $2 LIMIT 1
  `
	deletePipeSQL = `DELETE FROM pipes
    WHERE workspace_id = $1
    AND Key LIKE $2
  `
	insertPipesSQL = `
    WITH existing_pipe AS (
      UPDATE pipes SET data = $3
      WHERE workspace_id = $1 AND Key = $2
      RETURNING Key
    ),
    inserted_pipe AS (
      INSERT INTO pipes(workspace_id, Key, data)
      SELECT $1, $2, $3
      WHERE NOT EXISTS (SELECT 1 FROM existing_pipe)
      RETURNING Key
    )
    SELECT * FROM inserted_pipe
    UNION
    SELECT * FROM existing_pipe
  `
	deletePipeConnectionsSQL = `DELETE FROM connections
    WHERE workspace_id = $1
    AND Key = $2
  `
	selectPipesFromQueueSQL = `SELECT workspace_id, Key
	FROM get_queued_pipes()`

	queueAutomaticPipesSQL = `SELECT queue_automatic_pipes()`

	queuePipeAsFirstSQL = `SELECT queue_pipe_as_first($1, $2)`

	setQueuedPipeSyncedSQL = `UPDATE queued_pipes
	SET synced_at = now()
	WHERE workspace_id = $1
	AND Key = $2
	AND locked_at IS NOT NULL
	AND synced_at IS NULL`
)

const (
	selectPipeStatusSQL = `SELECT Key, data
    FROM pipes_status
    WHERE workspace_id = $1
  `
	singlePipeStatusSQL = `SELECT data
    FROM pipes_status
    WHERE workspace_id = $1
    AND Key = $2 LIMIT 1
  `
	deletePipeStatusSQL = `DELETE FROM pipes_status
		WHERE workspace_id = $1
		AND Key LIKE $2
  `
	lastSyncSQL = `SELECT (data->>'sync_date')::timestamp with time zone
    FROM pipes_status
    WHERE workspace_id = $1
    AND Key = $2
  `
	insertPipeStatusSQL = `
    WITH existing_status AS (
      UPDATE pipes_status SET data = $3
      WHERE workspace_id = $1 AND Key = $2
      RETURNING Key
    ),
    inserted_status AS (
      INSERT INTO pipes_status(workspace_id, Key, data)
      SELECT $1, $2, $3
      WHERE NOT EXISTS (SELECT 1 FROM existing_status)
      RETURNING Key
    )
    SELECT * FROM inserted_status
    UNION
    SELECT * FROM existing_status
  `
)
