package pipes

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bugsnag/bugsnag-go"

	"github.com/toggl/pipes-api/pkg/authorization"
	"github.com/toggl/pipes-api/pkg/connection"
	"github.com/toggl/pipes-api/pkg/environment"
	"github.com/toggl/pipes-api/pkg/integrations"
	"github.com/toggl/pipes-api/pkg/toggl"
)

const (
	usersPipeID    = "users"
	clientsPipeID  = "clients"
	projectsPipeID = "projects"
	tasksPipeId    = "tasks"
	todoPipeId     = "todolists"
)

type Service struct {
	env *environment.Environment
	api *toggl.ApiClient

	conn  *connection.Storage
	auth  *authorization.Storage
	pipes *Storage

	availablePipeType    *regexp.Regexp
	availableServiceType *regexp.Regexp
}

func NewService(env *environment.Environment, auth *authorization.Storage, pipes *Storage, conn *connection.Storage, api *toggl.ApiClient) *Service {
	svc := &Service{
		env: env,
		api: api,

		auth:  auth,
		pipes: pipes,
		conn:  conn,
	}

	svc.fillAvailablePipeTypes()
	svc.fillAvailableServices(env.GetIntegrations())
	return svc
}

func (svc *Service) postUsers(p *environment.PipeConfig) error {
	s, err := svc.auth.IntegrationFor(p)
	if err != nil {
		return err
	}
	usersResponse, err := svc.GetUsers(s)
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

	usersImport, err := svc.api.PostUsers(usersPipeID, toggl.UsersRequest{Users: users})
	if err != nil {
		return err
	}
	var connection *connection.Connection
	if connection, err = svc.conn.LoadConnection(s, usersPipeID); err != nil {
		return err
	}
	for _, user := range usersImport.WorkspaceUsers {
		connection.Data[user.ForeignID] = user.ID
	}
	if err := svc.conn.Save(connection); err != nil {
		return err
	}

	p.PipeStatus.Complete(usersPipeID, usersImport.Notifications, usersImport.Count())
	return nil
}

func (svc *Service) postClients(p *environment.PipeConfig) error {
	service, err := svc.auth.IntegrationFor(p)
	if err != nil {
		return err
	}
	clientsResponse, err := svc.getClients(service)
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
	clientsImport, err := svc.api.PostClients(clientsPipeID, clients)
	if err != nil {
		return err
	}
	var connection *connection.Connection
	if connection, err = svc.conn.LoadConnection(service, clientsPipeID); err != nil {
		return err
	}
	for _, client := range clientsImport.Clients {
		connection.Data[client.ForeignID] = client.ID
	}
	if err := svc.conn.Save(connection); err != nil {
		return err
	}
	p.PipeStatus.Complete(clientsPipeID, clientsImport.Notifications, clientsImport.Count())
	return nil
}

func (svc *Service) postProjects(p *environment.PipeConfig) error {
	s, err := svc.auth.IntegrationFor(p)
	if err != nil {
		return err
	}
	projectsResponse, err := svc.getProjects(s)
	if err != nil {
		return errors.New("unable to get projects from DB")
	}
	if projectsResponse == nil {
		return errors.New("service projects not found")
	}
	projects := toggl.ProjectRequest{
		Projects: projectsResponse.Projects,
	}
	projectsImport, err := svc.api.PostProjects(projectsPipeID, projects)
	if err != nil {
		return err
	}
	var connection *connection.Connection
	if connection, err = svc.conn.LoadConnection(s, projectsPipeID); err != nil {
		return err
	}
	for _, project := range projectsImport.Projects {
		connection.Data[project.ForeignID] = project.ID
	}
	if err := svc.conn.Save(connection); err != nil {
		return err
	}
	p.PipeStatus.Complete(projectsPipeID, projectsImport.Notifications, projectsImport.Count())
	return nil
}

func (svc *Service) postTodoLists(p *environment.PipeConfig) error {
	s, err := svc.auth.IntegrationFor(p)
	if err != nil {
		return err
	}
	tasksResponse, err := svc.getTasks(s, todoPipeId)
	if err != nil {
		return errors.New("unable to get tasks from DB")
	}
	if tasksResponse == nil {
		return errors.New("service tasks not found")
	}
	trs, err := toggl.AdjustRequestSize(tasksResponse.Tasks, 1)
	if err != nil {
		return err
	}
	var notifications []string
	var count int
	for _, tr := range trs {
		tasksImport, err := svc.api.PostTodoLists(tasksPipeId, tr)
		if err != nil {
			return err
		}
		connection, err := svc.conn.LoadConnection(s, todoPipeId)
		if err != nil {
			return err
		}
		for _, task := range tasksImport.Tasks {
			connection.Data[task.ForeignID] = task.ID
		}
		if err := svc.conn.Save(connection); err != nil {
			return err
		}
		notifications = append(notifications, tasksImport.Notifications...)
		count += tasksImport.Count()
	}
	p.PipeStatus.Complete(todoPipeId, notifications, count)
	return nil
}

func (svc *Service) postTasks(p *environment.PipeConfig) error {
	s, err := svc.auth.IntegrationFor(p)
	if err != nil {
		return err
	}
	tasksResponse, err := svc.getTasks(s, tasksPipeId)
	if err != nil {
		return errors.New("unable to get tasks from DB")
	}
	if tasksResponse == nil {
		return errors.New("service tasks not found")
	}
	trs, err := toggl.AdjustRequestSize(tasksResponse.Tasks, 1)
	if err != nil {
		return err
	}
	var notifications []string
	var count int
	for _, tr := range trs {
		tasksImport, err := svc.api.PostTasks(tasksPipeId, tr)
		if err != nil {
			return err
		}
		con, err := svc.conn.LoadConnection(s, tasksPipeId)
		if err != nil {
			return err
		}
		for _, task := range tasksImport.Tasks {
			con.Data[task.ForeignID] = task.ID
		}
		if err := svc.conn.Save(con); err != nil {
			return err
		}
		notifications = append(notifications, tasksImport.Notifications...)
		count += tasksImport.Count()
	}
	p.PipeStatus.Complete(p.ID, notifications, count)
	return nil
}

func (svc *Service) postTimeEntries(p *environment.PipeConfig, service integrations.Integration) error {
	var err error
	var entriesCon *connection.Connection
	var usersCon, tasksCon, projectsCon *connection.ReversedConnection
	if usersCon, err = svc.conn.LoadConnectionRev(service, "users"); err != nil {
		return err
	}
	if tasksCon, err = svc.conn.LoadConnectionRev(service, "tasks"); err != nil {
		return err
	}
	if projectsCon, err = svc.conn.LoadConnectionRev(service, "projects"); err != nil {
		return err
	}
	if entriesCon, err = svc.conn.LoadConnection(service, "time_entries"); err != nil {
		return err
	}

	if p.LastSync == nil {
		currentTime := time.Now()
		p.LastSync = &currentTime
	}

	timeEntries, err := svc.api.GetTimeEntries(*p.LastSync, usersCon.GetKeys(), projectsCon.GetKeys())
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

	if err := svc.conn.Save(entriesCon); err != nil {
		return err
	}
	p.PipeStatus.Complete("timeentries", []string{}, len(timeEntries))
	return nil
}

// ==========================  fetchSomething ==================================
func (svc *Service) fetchUsers(p *environment.PipeConfig) error {
	s, err := svc.auth.IntegrationFor(p)
	if err != nil {
		return err
	}
	users, err := s.Users()
	response := toggl.UsersResponse{Users: users}
	defer func() {
		s, err := svc.auth.IntegrationFor(p)
		if err != nil {
			log.Printf("could not get integration for pipe: %v, reason: %v", p.ID, err)
			return
		}
		workspaceID := s.GetWorkspaceID()
		objKey := s.KeyFor(usersPipeID)

		if err := svc.pipes.saveObject(workspaceID, objKey, response); err != nil {
			log.Printf("could not save object, workspaceID: %v key: %v, reason: %v", workspaceID, objKey, err)
			return
		}
	}()
	if err != nil {
		response.Error = err.Error()
		return err
	}
	return nil
}

func (svc *Service) fetchClients(p *environment.PipeConfig) error {
	s, err := svc.auth.IntegrationFor(p)
	if err != nil {
		return err
	}
	clients, err := s.Clients()
	response := toggl.ClientsResponse{Clients: clients}
	defer func() {
		s, err := svc.auth.IntegrationFor(p)
		if err != nil {
			log.Printf("could not get integration for pipe: %v, reason: %v", p.ID, err)
			return
		}
		workspaceID := s.GetWorkspaceID()
		objKey := s.KeyFor(clientsPipeID)

		if err := svc.pipes.saveObject(workspaceID, objKey, response); err != nil {
			log.Printf("could not save object, workspaceID: %v key: %v, reason: %v", workspaceID, objKey, err)
			return
		}
	}()
	if err != nil {
		response.Error = err.Error()
		return err
	}
	connections, err := svc.conn.LoadConnection(s, clientsPipeID)
	if err != nil {
		response.Error = err.Error()
		return err
	}
	for _, client := range response.Clients {
		client.ID = connections.Data[client.ForeignID]
	}
	return nil
}

func (svc *Service) fetchProjects(p *environment.PipeConfig) error {
	response := toggl.ProjectsResponse{}
	defer func() {
		s, err := svc.auth.IntegrationFor(p)
		if err != nil {
			log.Printf("could not get integration for pipe: %v, reason: %v", p.ID, err)
			return
		}
		workspaceID := s.GetWorkspaceID()
		objKey := s.KeyFor(projectsPipeID)

		if err := svc.pipes.saveObject(workspaceID, objKey, response); err != nil {
			log.Printf("could not save object, workspaceID: %v key: %v, reason: %v", workspaceID, objKey, err)
			return
		}
	}()

	if err := svc.fetchClients(p); err != nil {
		response.Error = err.Error()
		return err
	}
	if err := svc.postClients(p); err != nil {
		response.Error = err.Error()
		return err
	}

	service, err := svc.auth.IntegrationFor(p)
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

	var clientConnections, projectConnections *connection.Connection
	if clientConnections, err = svc.conn.LoadConnection(service, clientsPipeID); err != nil {
		response.Error = err.Error()
		return err
	}
	if projectConnections, err = svc.conn.LoadConnection(service, projectsPipeID); err != nil {
		response.Error = err.Error()
		return err
	}

	for _, project := range response.Projects {
		project.ID = projectConnections.Data[project.ForeignID]
		project.ClientID = clientConnections.Data[project.ForeignClientID]
	}

	return nil
}

func (svc *Service) fetchTodoLists(p *environment.PipeConfig) error {
	response := toggl.TasksResponse{}
	defer func() {
		s, err := svc.auth.IntegrationFor(p)
		if err != nil {
			log.Printf("could not get integration for pipe: %v, reason: %v", p.ID, err)
			return
		}
		workspaceID := s.GetWorkspaceID()
		objKey := s.KeyFor(todoPipeId)

		if err := svc.pipes.saveObject(workspaceID, objKey, response); err != nil {
			log.Printf("could not save object, workspaceID: %v key: %v, reason: %v", workspaceID, objKey, err)
			return
		}
	}()

	if err := svc.fetchProjects(p); err != nil {
		response.Error = err.Error()
		return err
	}
	if err := svc.postProjects(p); err != nil {
		response.Error = err.Error()
		return err
	}

	service, err := svc.auth.IntegrationFor(p)
	if err != nil {
		return err
	}
	service.SetSince(p.LastSync)
	tasks, err := service.TodoLists()
	if err != nil {
		response.Error = err.Error()
		return err
	}

	var projectConnections, taskConnections *connection.Connection

	if projectConnections, err = svc.conn.LoadConnection(service, projectsPipeID); err != nil {
		response.Error = err.Error()
		return err
	}
	if taskConnections, err = svc.conn.LoadConnection(service, todoPipeId); err != nil {
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

func (svc *Service) fetchTasks(p *environment.PipeConfig) error {
	response := toggl.TasksResponse{}
	defer func() {
		s, err := svc.auth.IntegrationFor(p)
		if err != nil {
			log.Printf("could not get integration for pipe: %v, reason: %v", p.ID, err)
			return
		}
		workspaceID := s.GetWorkspaceID()
		objKey := s.KeyFor(tasksPipeId)

		if err := svc.pipes.saveObject(workspaceID, objKey, response); err != nil {
			log.Printf("could not save object, workspaceID: %v key: %v, reason: %v", workspaceID, objKey, err)
			return
		}
	}()

	if err := svc.fetchProjects(p); err != nil {
		response.Error = err.Error()
		return err
	}
	if err := svc.postProjects(p); err != nil {
		response.Error = err.Error()
		return err
	}

	service, err := svc.auth.IntegrationFor(p)
	if err != nil {
		return err
	}
	service.SetSince(p.LastSync)
	tasks, err := service.Tasks()
	if err != nil {
		response.Error = err.Error()
		return err
	}
	var projectConnections, taskConnections *connection.Connection

	if projectConnections, err = svc.conn.LoadConnection(service, projectsPipeID); err != nil {
		response.Error = err.Error()
		return err
	}
	if taskConnections, err = svc.conn.LoadConnection(service, tasksPipeId); err != nil {
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

func (svc *Service) fetchTimeEntries(p *environment.PipeConfig) error {
	return nil
}

// =============================================================================

// ==========================  getSomething ====================================

func (svc *Service) GetUsers(s integrations.Integration) (*toggl.UsersResponse, error) {
	b, err := svc.pipes.getObject(s, usersPipeID)
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

func (svc *Service) getClients(s integrations.Integration) (*toggl.ClientsResponse, error) {
	b, err := svc.pipes.getObject(s, clientsPipeID)
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

func (svc *Service) getProjects(s integrations.Integration) (*toggl.ProjectsResponse, error) {
	b, err := svc.pipes.getObject(s, projectsPipeID)
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

func (svc *Service) getTasks(s integrations.Integration, objType string) (*toggl.TasksResponse, error) {
	b, err := svc.pipes.getObject(s, objType)
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

func (svc *Service) WorkspaceIntegrations(workspaceID int) ([]environment.IntegrationConfig, error) {
	authorizations, err := svc.auth.LoadAuthorizations(workspaceID)
	if err != nil {
		return nil, err
	}
	workspacePipes, err := svc.pipes.loadPipes(workspaceID)
	if err != nil {
		return nil, err
	}
	pipeStatuses, err := svc.pipes.loadPipeStatuses(workspaceID)
	if err != nil {
		return nil, err
	}

	var igr []environment.IntegrationConfig
	for _, current := range svc.env.GetIntegrations() {
		var integration = current
		integration.AuthURL = svc.env.OAuth2URL(integration.ID)
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

func (svc *Service) endSync(p *environment.PipeConfig, saveStatus bool, err error) error {
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
	if err = svc.pipes.savePipeStatus(p.PipeStatus); err != nil {
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

func (svc *Service) FetchObjects(p *environment.PipeConfig, saveStatus bool) (err error) {
	switch p.ID {
	case "users":
		err = svc.fetchUsers(p)
	case "projects":
		err = svc.fetchProjects(p)
	case "todolists":
		err = svc.fetchTodoLists(p)
	case "todos", "tasks":
		err = svc.fetchTasks(p)
	case "timeentries":
		err = svc.fetchTimeEntries(p)
	default:
		panic(fmt.Sprintf("FetchObjects: Unrecognized pipeID - %s", p.ID))
	}
	return svc.endSync(p, saveStatus, err)
}

func (svc *Service) postObjects(p *environment.PipeConfig, saveStatus bool) (err error) {
	switch p.ID {
	case "users":
		err = svc.postUsers(p)
	case "projects":
		err = svc.postProjects(p)
	case "todolists":
		err = svc.postTodoLists(p)
	case "todos", "tasks":
		err = svc.postTasks(p)
	case "timeentries":
		var service integrations.Integration
		service, err = svc.auth.IntegrationFor(p)
		if err != nil {
			break
		}
		err = svc.postTimeEntries(p, service)
	default:
		panic(fmt.Sprintf("postObjects: Unrecognized pipeID - %s", p.ID))
	}
	return svc.endSync(p, saveStatus, err)
}

func (svc *Service) newStatus(p *environment.PipeConfig) error {
	svc.pipes.loadLastSync(p)
	p.PipeStatus = environment.NewPipeStatus(p.WorkspaceID, p.ServiceID, p.ID, svc.env.GetPipesAPIHost())
	return svc.pipes.savePipeStatus(p.PipeStatus)
}

func (svc *Service) Run(p *environment.PipeConfig) {
	var err error
	defer func() {
		err := svc.endSync(p, true, err)
		log.Println(err)
	}()

	if err = svc.newStatus(p); err != nil {
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

	auth, err := svc.auth.LoadAuthFor(p)
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
	svc.api.WithAuthToken(auth.WorkspaceToken)

	if err = svc.FetchObjects(p, false); err != nil {
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
	if err = svc.postObjects(p, false); err != nil {
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

func (svc *Service) ClearPipeConnections(p *environment.PipeConfig) error {
	s, err := svc.auth.IntegrationFor(p)
	if err != nil {
		return err
	}

	key := s.KeyFor(p.ID)

	pipeStatus, err := svc.pipes.LoadPipeStatus(p.WorkspaceID, p.ServiceID, p.ID)
	if err != nil {
		return err
	}

	err = svc.pipes.DeletePipeConnections(p.WorkspaceID, key, pipeStatus.Key)
	if err != nil {
		return err
	}

	return nil
}

func (svc *Service) GetPipesFromQueue() ([]*environment.PipeConfig, error) {
	return svc.pipes.GetPipesFromQueue()
}

func (svc *Service) SetQueuedPipeSynced(pipe *environment.PipeConfig) error {
	return svc.pipes.SetQueuedPipeSynced(pipe)
}

func (svc *Service) QueueAutomaticPipes() error {
	return svc.pipes.QueueAutomaticPipes()
}

func (svc *Service) QueuePipeAsFirst(pipe *environment.PipeConfig) error {
	return svc.pipes.QueuePipeAsFirst(pipe)
}

func (svc *Service) AvailablePipeType(pipeID string) bool {
	return svc.availablePipeType.MatchString(pipeID)
}

func (svc *Service) AvailableServiceType(serviceID string) bool {
	return svc.availableServiceType.MatchString(serviceID)
}

func (svc *Service) fillAvailablePipeTypes() {
	svc.availablePipeType = regexp.MustCompile("users|projects|todolists|todos|tasks|timeentries")
}

func (svc *Service) fillAvailableServices(integrations []*environment.IntegrationConfig) {
	ids := make([]string, len(integrations))
	for i := range integrations {
		ids = append(ids, integrations[i].ID)
	}
	svc.availableServiceType = regexp.MustCompile(strings.Join(ids, "|"))
}
