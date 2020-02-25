package storage

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bugsnag/bugsnag-go"

	"github.com/toggl/pipes-api/pkg/environment"
	"github.com/toggl/pipes-api/pkg/integrations"
	"github.com/toggl/pipes-api/pkg/toggl"
)

type PipesStorage struct {
	AuthorizationStorage *AuthorizationStorage
	ConnectionStorage    *ConnectionStorage
	availablePipeType    *regexp.Regexp
	availableServiceType *regexp.Regexp

	env *environment.Environment
	db  *sql.DB
	api *toggl.ApiClient
}

func NewPipesStorage(env *environment.Environment, api *toggl.ApiClient, db *sql.DB) *PipesStorage {
	svc := &PipesStorage{
		AuthorizationStorage: &AuthorizationStorage{
			db:  db,
			env: env,
		},
		ConnectionStorage: &ConnectionStorage{
			db: db,
		},
		api: api,
		db:  db,
		env: env,
	}

	svc.fillAvailablePipeTypes()
	svc.fillAvailableServices(env.GetIntegrations())

	return svc
}

func (ps *PipesStorage) AvailablePipeType(pipeID string) bool {
	return ps.availablePipeType.MatchString(pipeID)
}

func (ps *PipesStorage) AvailableServiceType(serviceID string) bool {
	return ps.availableServiceType.MatchString(serviceID)
}

func (ps *PipesStorage) IsDown() bool {
	if _, err := ps.db.Exec("SELECT 1"); err != nil {
		return true
	}
	return false
}

func (ps *PipesStorage) LoadPipe(workspaceID int, serviceID, pipeID string) (*environment.PipeConfig, error) {
	key := environment.PipesKey(serviceID, pipeID)
	return ps.LoadPipeWithKey(workspaceID, key)
}

func (ps *PipesStorage) LoadPipeWithKey(workspaceID int, key string) (*environment.PipeConfig, error) {
	rows, err := ps.db.Query(singlePipesSQL, workspaceID, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, rows.Err()
	}
	var pipe environment.PipeConfig
	if err := ps.Load(rows, &pipe); err != nil {
		return nil, err
	}
	return &pipe, nil
}

func (ps *PipesStorage) LoadPipes(workspaceID int) (map[string]*environment.PipeConfig, error) {
	pipes := make(map[string]*environment.PipeConfig)
	rows, err := ps.db.Query(selectPipesSQL, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var pipe environment.PipeConfig
		if err := ps.Load(rows, &pipe); err != nil {
			return nil, err
		}
		pipes[pipe.Key] = &pipe
	}
	return pipes, nil
}

func (ps *PipesStorage) GetPipesFromQueue() ([]*environment.PipeConfig, error) {
	var pipes []*environment.PipeConfig
	rows, err := ps.db.Query(selectPipesFromQueueSQL)
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

func (ps *PipesStorage) SetQueuedPipeSynced(pipe *environment.PipeConfig) error {
	_, err := ps.db.Exec(setQueuedPipeSyncedSQL, pipe.WorkspaceID, pipe.Key)
	return err
}

func (ps *PipesStorage) Save(p *environment.PipeConfig) error {
	p.Configured = true
	b, err := json.Marshal(p)
	if err != nil {
		return err
	}
	_, err = ps.db.Exec(insertPipesSQL, p.WorkspaceID, p.Key, b)
	if err != nil {
		return err
	}
	return nil
}

func (ps *PipesStorage) Load(rows *sql.Rows, p *environment.PipeConfig) error {
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

func (ps *PipesStorage) NewStatus(p *environment.PipeConfig) error {
	ps.loadLastSync(p)
	p.PipeStatus = environment.NewPipeStatus(p.WorkspaceID, p.ServiceID, p.ID, ps.env.GetPipesAPIHost())
	return ps.savePipeStatus(p.PipeStatus)
}

func (ps *PipesStorage) Run(p *environment.PipeConfig) {
	var err error
	defer func() {
		err := ps.endSync(p, true, err)
		log.Println(err)
	}()

	if err = ps.NewStatus(p); err != nil {
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

	auth, err := ps.AuthorizationStorage.LoadAuthFor(p)
	if err != nil {
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
	ps.api.WithAuthToken(auth.WorkspaceToken)

	if err = ps.FetchObjects(p, false); err != nil {
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
	if err = ps.postObjects(p, false); err != nil {
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
}

func (ps *PipesStorage) loadLastSync(p *environment.PipeConfig) {
	err := ps.db.QueryRow(lastSyncSQL, p.WorkspaceID, p.Key).Scan(&p.LastSync)
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

func (ps *PipesStorage) FetchObjects(p *environment.PipeConfig, saveStatus bool) (err error) {
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

func (ps *PipesStorage) postObjects(p *environment.PipeConfig, saveStatus bool) (err error) {
	switch p.ID {
	case "users":
		err = ps.postUsers(p)
	case "projects":
		err = ps.postProjects(p)
	case "todolists":
		err = ps.postTodoLists(p)
	case "todos", "tasks":
		err = ps.postTasks(p)
	case "timeentries":
		var service integrations.Integration
		service, err = ps.AuthorizationStorage.IntegrationFor(p)
		if err != nil {
			break
		}
		err = ps.postTimeEntries(p, service)
	default:
		panic(fmt.Sprintf("postObjects: Unrecognized pipeID - %s", p.ID))
	}
	return ps.endSync(p, saveStatus, err)
}

func (ps *PipesStorage) Destroy(p *environment.PipeConfig, workspaceID int) error {
	tx, err := ps.db.Begin()
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

func (ps *PipesStorage) ClearPipeConnections(p *environment.PipeConfig) (err error) {
	s, err := ps.AuthorizationStorage.IntegrationFor(p)
	if err != nil {
		return
	}

	pipeStatus, err := ps.LoadPipeStatus(p.WorkspaceID, p.ServiceID, p.ID)
	if err != nil {
		return
	}

	key := s.KeyFor(p.ID)

	tx, err := ps.db.Begin()
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

func (ps *PipesStorage) LoadPipeStatus(workspaceID int, serviceID, pipeID string) (*environment.PipeStatusConfig, error) {
	key := environment.PipesKey(serviceID, pipeID)
	rows, err := ps.db.Query(singlePipeStatusSQL, workspaceID, key)
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
	var pipeStatus environment.PipeStatusConfig
	if err = json.Unmarshal(b, &pipeStatus); err != nil {
		return nil, err
	}
	pipeStatus.WorkspaceID = workspaceID
	pipeStatus.ServiceID = serviceID
	pipeStatus.PipeID = pipeID
	return &pipeStatus, nil
}

func (ps *PipesStorage) loadPipeStatuses(workspaceID int) (map[string]*environment.PipeStatusConfig, error) {
	pipeStatuses := make(map[string]*environment.PipeStatusConfig)
	rows, err := ps.db.Query(selectPipeStatusSQL, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var pipeStatus environment.PipeStatusConfig
		var b []byte
		var key string
		if err := rows.Scan(&key, &b); err != nil {
			return nil, err
		}
		err := json.Unmarshal(b, &pipeStatus)
		if err != nil {
			return nil, err
		}
		pipeStatus.Key = key
		pipeStatuses[pipeStatus.Key] = &pipeStatus
	}
	return pipeStatuses, nil
}

func (ps *PipesStorage) savePipeStatus(p *environment.PipeStatusConfig) error {
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
	_, err = ps.db.Exec(insertPipeStatusSQL, p.WorkspaceID, p.Key, b)
	if err != nil {
		return err
	}
	return nil
}

func (ps *PipesStorage) endSync(p *environment.PipeConfig, saveStatus bool, err error) error {
	if !saveStatus {
		return err
	}

	if err != nil {
		// If it is JSON marshalling error suppress it for status
		if _, ok := err.(*json.UnmarshalTypeError); ok {
			err = environment.ErrJSONParsing
		}
		p.PipeStatus.AddError(err)
	}
	if err = ps.savePipeStatus(p.PipeStatus); err != nil {
		bugsnag.Notify(err, bugsnag.MetaData{
			"pipe": {
				"ID":            p.ID,
				"Name":          p.Name,
				"ServiceParams": string(p.ServiceParams),
				"WorkspaceID":   p.WorkspaceID,
				"ServiceID":     p.ServiceID,
			},
		})
		return err
	}

	return nil
}

func (ps *PipesStorage) WorkspaceIntegrations(workspaceID int) ([]environment.IntegrationConfig, error) {
	authorizations, err := ps.AuthorizationStorage.LoadAuthorizations(workspaceID)
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

	var igr []environment.IntegrationConfig
	for _, current := range ps.env.GetIntegrations() {
		var integration = current
		integration.AuthURL = ps.env.OAuth2URL(integration.ID)
		integration.Authorized = authorizations[integration.ID]
		var pipes []*environment.PipeConfig
		for i := range integration.Pipes {
			var pipe = *integration.Pipes[i]
			key := environment.PipesKey(integration.ID, pipe.ID)
			existingPipe := workspacePipes[key]
			if existingPipe != nil {
				pipe.Automatic = existingPipe.Automatic
				pipe.Configured = existingPipe.Configured
			}

			pipe.PipeStatus = pipeStatuses[key]
			pipes = append(pipes, &pipe)
		}
		integration.Pipes = pipes
		igr = append(igr, *integration)
	}
	return igr, nil
}

func (ps *PipesStorage) QueueAutomaticPipes() error {
	_, err := ps.db.Exec(queueAutomaticPipesSQL)
	return err
}

func (ps *PipesStorage) DeletePipeByWorkspaceIDServiceID(workspaceID int, serviceID string) error {
	_, err := ps.db.Exec(deletePipeSQL, workspaceID, serviceID+"%")
	return err
}

func (ps *PipesStorage) QueuePipeAsFirst(pipe *environment.PipeConfig) error {
	_, err := ps.db.Exec(queuePipeAsFirstSQL, pipe.WorkspaceID, pipe.Key)
	return err
}

func (ps *PipesStorage) GetAccounts(s integrations.Integration) (*toggl.AccountsResponse, error) {
	var result []byte
	rows, err := ps.db.Query(`
		SELECT data FROM imports
		WHERE workspace_id = $1 AND Key = $2
		ORDER by created_at DESC
		LIMIT 1
	`, s.GetWorkspaceID(), s.KeyFor("accounts"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, rows.Err()
	}
	if err := rows.Scan(&result); err != nil {
		return nil, err
	}

	var accountsResponse toggl.AccountsResponse
	err = json.Unmarshal(result, &accountsResponse)
	if err != nil {
		return nil, err
	}
	return &accountsResponse, nil
}

func (ps *PipesStorage) FetchAccounts(s integrations.Integration) error {
	var response toggl.AccountsResponse
	accounts, err := s.Accounts()
	response.Accounts = accounts
	if err != nil {
		response.Error = err.Error()
	}

	b, err := json.Marshal(response)
	if err != nil {
		bugsnag.Notify(err)
		return err
	}
	_, err = ps.db.Exec(`
    INSERT INTO imports(workspace_id, Key, data, created_at)
    VALUES($1, $2, $3, NOW())
  	`, s.GetWorkspaceID(), s.KeyFor("accounts"), b)
	if err != nil {
		bugsnag.Notify(err)
		return err
	}
	return nil
}

func (ps *PipesStorage) ClearImportFor(s integrations.Integration, pipeID string) error {
	_, err := ps.db.Exec(`
	    DELETE FROM imports
	    WHERE workspace_id = $1 AND Key = $2
	`, s.GetWorkspaceID(), s.KeyFor(pipeID))
	return err
}

// ==========================  postSomething ===================================
func (ps *PipesStorage) postUsers(p *environment.PipeConfig) error {
	s, err := ps.AuthorizationStorage.IntegrationFor(p)
	if err != nil {
		return err
	}
	usersResponse, err := ps.GetUsers(s)
	if err != nil {
		return errors.New("unable to get users from DB")
	}
	if usersResponse == nil {
		return errors.New("service users not found")
	}

	var selector Selector
	if err := json.Unmarshal(p.Payload, &selector); err != nil {
		return err
	}

	var users []*toggl.User
	for _, userID := range selector.IDs {
		for _, user := range usersResponse.Users {
			if user.ForeignID == strconv.Itoa(userID) {
				user.SendInvitation = selector.SendInvites
				users = append(users, user)
			}
		}
	}

	usersImport, err := ps.api.PostUsers(usersPipeID, toggl.UsersRequest{Users: users})
	if err != nil {
		return err
	}
	var connection *Connection
	if connection, err = ps.ConnectionStorage.LoadConnection(s, usersPipeID); err != nil {
		return err
	}
	for _, user := range usersImport.WorkspaceUsers {
		connection.Data[user.ForeignID] = user.ID
	}
	if err := ps.ConnectionStorage.Save(connection); err != nil {
		return err
	}

	p.PipeStatus.Complete(usersPipeID, usersImport.Notifications, usersImport.Count())
	return nil
}

func (ps *PipesStorage) postClients(p *environment.PipeConfig) error {
	service, err := ps.AuthorizationStorage.IntegrationFor(p)
	if err != nil {
		return err
	}
	clientsResponse, err := ps.getClients(service)
	if err != nil {
		return errors.New("unable to get clients from DB")
	}
	if clientsResponse == nil {
		return errors.New("service clients not found")
	}
	clients := toggl.ClientRequest{
		Clients: clientsResponse.Clients,
	}
	if len(clientsResponse.Clients) == 0 {
		return nil
	}
	clientsImport, err := ps.api.PostClients(clientsPipeID, clients)
	if err != nil {
		return err
	}
	var connection *Connection
	if connection, err = ps.ConnectionStorage.LoadConnection(service, clientsPipeID); err != nil {
		return err
	}
	for _, client := range clientsImport.Clients {
		connection.Data[client.ForeignID] = client.ID
	}
	if err := ps.ConnectionStorage.Save(connection); err != nil {
		return err
	}
	p.PipeStatus.Complete(clientsPipeID, clientsImport.Notifications, clientsImport.Count())
	return nil
}

func (ps *PipesStorage) postProjects(p *environment.PipeConfig) error {
	s, err := ps.AuthorizationStorage.IntegrationFor(p)
	if err != nil {
		return err
	}
	projectsResponse, err := ps.getProjects(s)
	if err != nil {
		return errors.New("unable to get projects from DB")
	}
	if projectsResponse == nil {
		return errors.New("service projects not found")
	}
	projects := toggl.ProjectRequest{
		Projects: projectsResponse.Projects,
	}
	projectsImport, err := ps.api.PostProjects(projectsPipeID, projects)
	if err != nil {
		return err
	}
	var connection *Connection
	if connection, err = ps.ConnectionStorage.LoadConnection(s, projectsPipeID); err != nil {
		return err
	}
	for _, project := range projectsImport.Projects {
		connection.Data[project.ForeignID] = project.ID
	}
	if err := ps.ConnectionStorage.Save(connection); err != nil {
		return err
	}
	p.PipeStatus.Complete(projectsPipeID, projectsImport.Notifications, projectsImport.Count())
	return nil
}

func (ps *PipesStorage) postTodoLists(p *environment.PipeConfig) error {
	s, err := ps.AuthorizationStorage.IntegrationFor(p)
	if err != nil {
		return err
	}
	tasksResponse, err := ps.getTasks(s, todoPipeId)
	if err != nil {
		return errors.New("unable to get tasks from DB")
	}
	if tasksResponse == nil {
		return errors.New("service tasks not found")
	}
	trs, err := adjustRequestSize(tasksResponse.Tasks, 1)
	if err != nil {
		return err
	}
	var notifications []string
	var count int
	for _, tr := range trs {
		tasksImport, err := ps.api.PostTodoLists(tasksPipeId, tr)
		if err != nil {
			return err
		}
		connection, err := ps.ConnectionStorage.LoadConnection(s, todoPipeId)
		if err != nil {
			return err
		}
		for _, task := range tasksImport.Tasks {
			connection.Data[task.ForeignID] = task.ID
		}
		if err := ps.ConnectionStorage.Save(connection); err != nil {
			return err
		}
		notifications = append(notifications, tasksImport.Notifications...)
		count += tasksImport.Count()
	}
	p.PipeStatus.Complete(todoPipeId, notifications, count)
	return nil
}

func (ps *PipesStorage) postTasks(p *environment.PipeConfig) error {
	s, err := ps.AuthorizationStorage.IntegrationFor(p)
	if err != nil {
		return err
	}
	tasksResponse, err := ps.getTasks(s, tasksPipeId)
	if err != nil {
		return errors.New("unable to get tasks from DB")
	}
	if tasksResponse == nil {
		return errors.New("service tasks not found")
	}
	trs, err := adjustRequestSize(tasksResponse.Tasks, 1)
	if err != nil {
		return err
	}
	var notifications []string
	var count int
	for _, tr := range trs {
		tasksImport, err := ps.api.PostTasks(tasksPipeId, tr)
		if err != nil {
			return err
		}
		connection, err := ps.ConnectionStorage.LoadConnection(s, tasksPipeId)
		if err != nil {
			return err
		}
		for _, task := range tasksImport.Tasks {
			connection.Data[task.ForeignID] = task.ID
		}
		if err := ps.ConnectionStorage.Save(connection); err != nil {
			return err
		}
		notifications = append(notifications, tasksImport.Notifications...)
		count += tasksImport.Count()
	}
	p.PipeStatus.Complete(p.ID, notifications, count)
	return nil
}

func (ps *PipesStorage) postTimeEntries(p *environment.PipeConfig, service integrations.Integration) error {
	var err error
	var entriesCon *Connection
	var usersCon, tasksCon, projectsCon *ReversedConnection
	if usersCon, err = ps.ConnectionStorage.LoadConnectionRev(service, "users"); err != nil {
		return err
	}
	if tasksCon, err = ps.ConnectionStorage.LoadConnectionRev(service, "tasks"); err != nil {
		return err
	}
	if projectsCon, err = ps.ConnectionStorage.LoadConnectionRev(service, "projects"); err != nil {
		return err
	}
	if entriesCon, err = ps.ConnectionStorage.LoadConnection(service, "time_entries"); err != nil {
		return err
	}

	if p.LastSync == nil {
		currentTime := time.Now()
		p.LastSync = &currentTime
	}

	timeEntries, err := ps.api.GetTimeEntries(*p.LastSync, usersCon.GetKeys(), projectsCon.GetKeys())
	if err != nil {
		return err
	}

	for _, entry := range timeEntries {
		entry.ForeignID = strconv.Itoa(entriesCon.Data[strconv.Itoa(entry.ID)])
		entry.ForeignTaskID = strconv.Itoa(tasksCon.GetInt(entry.TaskID))
		entry.ForeignUserID = strconv.Itoa(usersCon.GetInt(entry.UserID))
		entry.ForeignProjectID = strconv.Itoa(projectsCon.GetInt(entry.ProjectID))

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

	if err := ps.ConnectionStorage.Save(entriesCon); err != nil {
		return err
	}
	p.PipeStatus.Complete("timeentries", []string{}, len(timeEntries))
	return nil
}

// =============================================================================

// ==========================  fetchSomething ==================================
func (ps *PipesStorage) fetchUsers(p *environment.PipeConfig) error {
	s, err := ps.AuthorizationStorage.IntegrationFor(p)
	if err != nil {
		return err
	}
	users, err := s.Users()
	response := toggl.UsersResponse{Users: users}
	defer func() { ps.saveObject(p, usersPipeID, response) }()
	if err != nil {
		response.Error = err.Error()
		return err
	}
	return nil
}

func (ps *PipesStorage) fetchClients(p *environment.PipeConfig) error {
	s, err := ps.AuthorizationStorage.IntegrationFor(p)
	if err != nil {
		return err
	}
	clients, err := s.Clients()
	response := toggl.ClientsResponse{Clients: clients}
	defer func() { ps.saveObject(p, clientsPipeID, response) }()
	if err != nil {
		response.Error = err.Error()
		return err
	}
	connections, err := ps.ConnectionStorage.LoadConnection(s, clientsPipeID)
	if err != nil {
		response.Error = err.Error()
		return err
	}
	for _, client := range response.Clients {
		client.ID = connections.Data[client.ForeignID]
	}
	return nil
}

func (ps *PipesStorage) fetchProjects(p *environment.PipeConfig) error {
	response := toggl.ProjectsResponse{}
	defer func() { ps.saveObject(p, projectsPipeID, response) }()

	if err := ps.fetchClients(p); err != nil {
		response.Error = err.Error()
		return err
	}
	if err := ps.postClients(p); err != nil {
		response.Error = err.Error()
		return err
	}

	service, err := ps.AuthorizationStorage.IntegrationFor(p)
	if err != nil {
		return err
	}
	service.SetSince(p.LastSync)
	projects, err := service.Projects()
	if err != nil {
		response.Error = err.Error()
		return err
	}

	response.Projects = trimSpacesFromName(projects)

	var clientConnections, projectConnections *Connection
	if clientConnections, err = ps.ConnectionStorage.LoadConnection(service, clientsPipeID); err != nil {
		response.Error = err.Error()
		return err
	}
	if projectConnections, err = ps.ConnectionStorage.LoadConnection(service, projectsPipeID); err != nil {
		response.Error = err.Error()
		return err
	}

	for _, project := range response.Projects {
		project.ID = projectConnections.Data[project.ForeignID]
		project.ClientID = clientConnections.Data[project.ForeignClientID]
	}

	return nil
}

func (ps *PipesStorage) fetchTodoLists(p *environment.PipeConfig) error {
	response := toggl.TasksResponse{}
	defer func() { ps.saveObject(p, todoPipeId, response) }()

	if err := ps.fetchProjects(p); err != nil {
		response.Error = err.Error()
		return err
	}
	if err := ps.postProjects(p); err != nil {
		response.Error = err.Error()
		return err
	}

	service, err := ps.AuthorizationStorage.IntegrationFor(p)
	if err != nil {
		return err
	}
	service.SetSince(p.LastSync)
	tasks, err := service.TodoLists()
	if err != nil {
		response.Error = err.Error()
		return err
	}

	var projectConnections, taskConnections *Connection

	if projectConnections, err = ps.ConnectionStorage.LoadConnection(service, projectsPipeID); err != nil {
		response.Error = err.Error()
		return err
	}
	if taskConnections, err = ps.ConnectionStorage.LoadConnection(service, todoPipeId); err != nil {
		response.Error = err.Error()
		return err
	}

	response.Tasks = make([]*toggl.Task, 0)
	for _, task := range tasks {
		id := taskConnections.Data[task.ForeignID]
		if (id > 0) || task.Active {
			task.ID = id
			task.ProjectID = projectConnections.Data[task.ForeignProjectID]
			response.Tasks = append(response.Tasks, task)
		}
	}
	return nil
}

func (ps *PipesStorage) fetchTasks(p *environment.PipeConfig) error {
	response := toggl.TasksResponse{}
	defer func() { ps.saveObject(p, tasksPipeId, response) }()

	if err := ps.fetchProjects(p); err != nil {
		response.Error = err.Error()
		return err
	}
	if err := ps.postProjects(p); err != nil {
		response.Error = err.Error()
		return err
	}

	service, err := ps.AuthorizationStorage.IntegrationFor(p)
	if err != nil {
		return err
	}
	service.SetSince(p.LastSync)
	tasks, err := service.Tasks()
	if err != nil {
		response.Error = err.Error()
		return err
	}
	var projectConnections, taskConnections *Connection

	if projectConnections, err = ps.ConnectionStorage.LoadConnection(service, projectsPipeID); err != nil {
		response.Error = err.Error()
		return err
	}
	if taskConnections, err = ps.ConnectionStorage.LoadConnection(service, tasksPipeId); err != nil {
		response.Error = err.Error()
		return err
	}

	response.Tasks = make([]*toggl.Task, 0)
	for _, task := range tasks {
		id := taskConnections.Data[task.ForeignID]
		if (id > 0) || task.Active {
			task.ID = id
			task.ProjectID = projectConnections.Data[task.ForeignProjectID]
			response.Tasks = append(response.Tasks, task)
		}
	}
	return nil
}

func (ps *PipesStorage) fetchTimeEntries(p *environment.PipeConfig) error {
	return nil
}

// =============================================================================

// ==========================  getSomething ====================================

func (ps *PipesStorage) GetUsers(s integrations.Integration) (*toggl.UsersResponse, error) {
	b, err := ps.getObject(s, usersPipeID)
	if err != nil || b == nil {
		return nil, err
	}

	var usersResponse toggl.UsersResponse
	err = json.Unmarshal(b, &usersResponse)
	if err != nil {
		return nil, err
	}
	return &usersResponse, nil
}

func (ps *PipesStorage) getClients(s integrations.Integration) (*toggl.ClientsResponse, error) {
	b, err := ps.getObject(s, clientsPipeID)
	if err != nil || b == nil {
		return nil, err
	}

	var clientsResponse toggl.ClientsResponse
	err = json.Unmarshal(b, &clientsResponse)
	if err != nil {
		return nil, err
	}
	return &clientsResponse, nil
}

func (ps *PipesStorage) getProjects(s integrations.Integration) (*toggl.ProjectsResponse, error) {
	b, err := ps.getObject(s, projectsPipeID)
	if err != nil || b == nil {
		return nil, err
	}

	var projectsResponse toggl.ProjectsResponse
	err = json.Unmarshal(b, &projectsResponse)
	if err != nil {
		return nil, err
	}

	return &projectsResponse, nil
}

func (ps *PipesStorage) getTasks(s integrations.Integration, objType string) (*toggl.TasksResponse, error) {
	b, err := ps.getObject(s, objType)
	if err != nil || b == nil {
		return nil, err
	}

	var tasksResponse toggl.TasksResponse
	err = json.Unmarshal(b, &tasksResponse)
	if err != nil {
		return nil, err
	}
	return &tasksResponse, nil
}

// =============================================================================

// ========================== get/saveObject ===================================
func (ps *PipesStorage) getObject(s integrations.Integration, pipeID string) ([]byte, error) {
	var result []byte
	rows, err := ps.db.Query(`
		SELECT data FROM imports
		WHERE workspace_id = $1 AND Key = $2
		ORDER by created_at DESC
		LIMIT 1
	`, s.GetWorkspaceID(), s.KeyFor(pipeID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, rows.Err()
	}
	if err := rows.Scan(&result); err != nil {
		return nil, err
	}
	return result, nil
}

func (ps *PipesStorage) saveObject(p *environment.PipeConfig, pipeID string, obj interface{}) error {
	b, err := json.Marshal(obj)
	if err != nil {
		bugsnag.Notify(err)
		return err
	}
	s, err := ps.AuthorizationStorage.IntegrationFor(p)
	if err != nil {
		return err
	}
	_, err = ps.db.Exec(`
	  INSERT INTO imports(workspace_id, Key, data, created_at)
    VALUES($1, $2, $3, NOW())
	`, p.WorkspaceID, s.KeyFor(pipeID), b)
	if err != nil {
		bugsnag.Notify(err)
		return err
	}
	return nil
}

// =============================================================================

func (ps *PipesStorage) fillAvailablePipeTypes() {
	ps.availablePipeType = regexp.MustCompile("users|projects|todolists|todos|tasks|timeentries")
}

func (ps *PipesStorage) fillAvailableServices(integrations []*environment.IntegrationConfig) {
	ids := make([]string, len(integrations))
	for i := range integrations {
		ids = append(ids, integrations[i].ID)
	}
	ps.availableServiceType = regexp.MustCompile(strings.Join(ids, "|"))
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
