package integrations

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bugsnag/bugsnag-go"

	"github.com/toggl/pipes-api/pkg/authorization"
	"github.com/toggl/pipes-api/pkg/connection"
	"github.com/toggl/pipes-api/pkg/toggl"
)

type OauthProvider interface {
	GetOAuth2URL(integrationID string) string
}

type Service struct {
	api   *toggl.ApiClient
	oauth OauthProvider

	conn  *connection.Storage
	auth  *authorization.Storage
	pipes *Storage

	availablePipeType    *regexp.Regexp
	availableServiceType *regexp.Regexp

	availableIntegrations []*Integration
	pipesApiHost          string
	mx                    sync.RWMutex
}

func NewService(oauth OauthProvider, auth *authorization.Storage, pipes *Storage, conn *connection.Storage, api *toggl.ApiClient, pipesApiHost, workDir string) *Service {
	svc := &Service{
		api:   api,
		oauth: oauth,
		auth:  auth,
		pipes: pipes,
		conn:  conn,

		pipesApiHost:          pipesApiHost,
		availableIntegrations: []*Integration{},
	}

	svc.loadIntegrations(workDir).
		fillAvailableServices().
		fillAvailablePipeTypes()

	svc.mx.RLock()
	for _, integration := range svc.availableIntegrations {
		svc.auth.SetAuthorizationType(integration.ID, integration.AuthType)
	}
	svc.mx.RUnlock()

	return svc
}

func (svc *Service) GetUsers(s ExternalService) (*toggl.UsersResponse, error) {
	b, err := svc.pipes.getObject(s, toggl.UsersPipeID)
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

func (svc *Service) WorkspaceIntegrations(workspaceID int) ([]Integration, error) {
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

	var igr []Integration
	for _, current := range svc.getIntegrations() {
		var integration = current
		integration.AuthURL = svc.oauth.GetOAuth2URL(integration.ID)
		integration.Authorized = authorizations[integration.ID]
		var pipes []*Pipe
		for i := range integration.Pipes {
			var pipe = *integration.Pipes[i]
			key := PipesKey(integration.ID, pipe.ID)
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

func (svc *Service) FetchObjects(p *Pipe, saveStatus bool) (err error) {
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

func (svc *Service) Run(p *Pipe) {
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

	s := Create(p.ServiceID, p.WorkspaceID)
	auth, err := svc.auth.LoadAuth(s.GetWorkspaceID(), s.Name())
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

	if err := s.SetAuthData(auth.Data); err != nil {
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

	if err = svc.auth.Refresh(auth); err != nil {
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

func (svc *Service) ClearPipeConnections(p *Pipe) error {

	service := Create(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}
	auth, err := svc.auth.LoadAuth(service.GetWorkspaceID(), service.Name())
	if err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}

	key := service.KeyFor(p.ID)

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

func (svc *Service) Ready() []error {
	errs := make([]error, 0)

	if svc.pipes.IsDown() {
		errs = append(errs, errors.New("database is down"))
	}

	if err := svc.api.Ping(); err != nil {
		errs = append(errs, err)
	}
	return errs
}

func (svc *Service) GetPipesFromQueue() ([]*Pipe, error) {
	return svc.pipes.GetPipesFromQueue()
}

func (svc *Service) SetQueuedPipeSynced(pipe *Pipe) error {
	return svc.pipes.SetQueuedPipeSynced(pipe)
}

func (svc *Service) QueueAutomaticPipes() error {
	return svc.pipes.QueueAutomaticPipes()
}

func (svc *Service) QueuePipeAsFirst(pipe *Pipe) error {
	return svc.pipes.QueuePipeAsFirst(pipe)
}

func (svc *Service) AvailablePipeType(pipeID string) bool {
	return svc.availablePipeType.MatchString(pipeID)
}

func (svc *Service) AvailableServiceType(serviceID string) bool {
	return svc.availableServiceType.MatchString(serviceID)
}

func (svc *Service) loadIntegrations(workDir string) *Service {
	svc.mx.Lock()
	defer svc.mx.Unlock()
	b, err := ioutil.ReadFile(filepath.Join(workDir, "config", "integrations.json"))
	if err != nil {
		log.Fatalf("Could not read integrations.json, reason: %v", err)
	}
	if err := json.Unmarshal(b, &svc.availableIntegrations); err != nil {
		log.Fatalf("Could not parse integrations.json, reason: %v", err)
	}
	return svc
}

func (svc *Service) getIntegrations() []*Integration {
	svc.mx.RLock()
	defer svc.mx.RUnlock()
	return svc.availableIntegrations
}

func (svc *Service) fillAvailableServices() *Service {
	ids := make([]string, len(svc.availableIntegrations))
	for i := range svc.availableIntegrations {
		ids = append(ids, svc.availableIntegrations[i].ID)
	}
	svc.availableServiceType = regexp.MustCompile(strings.Join(ids, "|"))
	return svc
}

func (svc *Service) postUsers(p *Pipe) error {
	service := Create(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}

	auth, err := svc.auth.LoadAuth(service.GetWorkspaceID(), service.Name())
	if err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}

	usersResponse, err := svc.GetUsers(service)
	if err != nil {
		return errors.New("unable to get users from DB")
	}
	if usersResponse == nil {
		return errors.New("service users not found")
	}

	var selector struct {
		IDs         []int `json:"ids"`
		SendInvites bool  `json:"send_invites"`
	}
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

	usersImport, err := svc.api.PostUsers(toggl.UsersPipeID, toggl.UsersRequest{Users: users})
	if err != nil {
		return err
	}
	var connection *connection.Connection
	if connection, err = svc.conn.Load(service.GetWorkspaceID(), service.KeyFor(toggl.UsersPipeID)); err != nil {
		return err
	}
	for _, user := range usersImport.WorkspaceUsers {
		connection.Data[user.ForeignID] = user.ID
	}
	if err := svc.conn.Save(connection); err != nil {
		return err
	}

	p.PipeStatus.Complete(toggl.UsersPipeID, usersImport.Notifications, usersImport.Count())
	return nil
}

func (svc *Service) postClients(p *Pipe) error {
	service := Create(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}
	auth, err := svc.auth.LoadAuth(service.GetWorkspaceID(), service.Name())
	if err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
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
	clientsImport, err := svc.api.PostClients(toggl.ClientsPipeID, clients)
	if err != nil {
		return err
	}
	var connection *connection.Connection
	if connection, err = svc.conn.Load(service.GetWorkspaceID(), service.KeyFor(toggl.ClientsPipeID)); err != nil {
		return err
	}
	for _, client := range clientsImport.Clients {
		connection.Data[client.ForeignID] = client.ID
	}
	if err := svc.conn.Save(connection); err != nil {
		return err
	}
	p.PipeStatus.Complete(toggl.ClientsPipeID, clientsImport.Notifications, clientsImport.Count())
	return nil
}

func (svc *Service) postProjects(p *Pipe) error {
	service := Create(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}

	auth, err := svc.auth.LoadAuth(service.GetWorkspaceID(), service.Name())
	if err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}

	projectsResponse, err := svc.getProjects(service)
	if err != nil {
		return errors.New("unable to get projects from DB")
	}
	if projectsResponse == nil {
		return errors.New("service projects not found")
	}
	projects := toggl.ProjectRequest{
		Projects: projectsResponse.Projects,
	}
	projectsImport, err := svc.api.PostProjects(toggl.ProjectsPipeID, projects)
	if err != nil {
		return err
	}
	var connection *connection.Connection
	if connection, err = svc.conn.Load(service.GetWorkspaceID(), service.KeyFor(toggl.ProjectsPipeID)); err != nil {
		return err
	}
	for _, project := range projectsImport.Projects {
		connection.Data[project.ForeignID] = project.ID
	}
	if err := svc.conn.Save(connection); err != nil {
		return err
	}
	p.PipeStatus.Complete(toggl.ProjectsPipeID, projectsImport.Notifications, projectsImport.Count())
	return nil
}

func (svc *Service) postTodoLists(p *Pipe) error {
	service := Create(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}

	auth, err := svc.auth.LoadAuth(service.GetWorkspaceID(), service.Name())
	if err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}

	tasksResponse, err := svc.getTasks(service, toggl.TodoPipeID) // TODO: WTF?? Why toggl.TodoPipeID
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
		tasksImport, err := svc.api.PostTodoLists(toggl.TasksPipeID, tr) // TODO: WTF?? Why toggl.TasksPipeID
		if err != nil {
			return err
		}
		connection, err := svc.conn.Load(service.GetWorkspaceID(), service.KeyFor(toggl.TodoPipeID))
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
	p.PipeStatus.Complete(toggl.TodoPipeID, notifications, count)
	return nil
}

func (svc *Service) postTasks(p *Pipe) error {
	service := Create(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}
	auth, err := svc.auth.LoadAuth(service.GetWorkspaceID(), service.Name())
	if err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}

	tasksResponse, err := svc.getTasks(service, toggl.TasksPipeID)
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
		tasksImport, err := svc.api.PostTasks(toggl.TasksPipeID, tr)
		if err != nil {
			return err
		}
		con, err := svc.conn.Load(service.GetWorkspaceID(), service.KeyFor(toggl.TasksPipeID))
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

func (svc *Service) postTimeEntries(p *Pipe, service ExternalService) error {
	usersCon, err := svc.conn.LoadReversed(service.GetWorkspaceID(), service.KeyFor(toggl.UsersPipeID))
	if err != nil {
		return err
	}

	tasksCon, err := svc.conn.LoadReversed(service.GetWorkspaceID(), service.KeyFor(toggl.TasksPipeID))
	if err != nil {
		return err
	}

	projectsCon, err := svc.conn.LoadReversed(service.GetWorkspaceID(), service.KeyFor(toggl.ProjectsPipeID))
	if err != nil {
		return err
	}

	entriesCon, err := svc.conn.Load(service.GetWorkspaceID(), service.KeyFor(toggl.TimeEntriesPipeID))
	if err != nil {
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
		entry.ForeignTaskID = strconv.Itoa(tasksCon.GetForeignID(entry.TaskID))
		entry.ForeignUserID = strconv.Itoa(usersCon.GetForeignID(entry.UserID))
		entry.ForeignProjectID = strconv.Itoa(projectsCon.GetForeignID(entry.ProjectID))

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
func (svc *Service) fetchUsers(p *Pipe) error {
	service := Create(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}

	auth, err := svc.auth.LoadAuth(service.GetWorkspaceID(), service.Name())
	if err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}

	users, err := service.Users()
	response := toggl.UsersResponse{Users: users}
	defer func() {
		if err := service.SetParams(p.ServiceParams); err != nil {
			log.Printf("could not set service params: %v, reason: %v", p.ID, err)
			return
		}
		auth, err := svc.auth.LoadAuth(service.GetWorkspaceID(), service.Name())
		if err != nil {
			log.Printf("could not get service auth: %v, reason: %v", p.ID, err)
			return
		}
		if err := service.SetAuthData(auth.Data); err != nil {
			log.Printf("could not set auth data: %v, reason: %v", p.ID, err)
			return
		}

		workspaceID := service.GetWorkspaceID()
		objKey := service.KeyFor(toggl.UsersPipeID)

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

func (svc *Service) fetchClients(p *Pipe) error {
	service := Create(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}
	auth, err := svc.auth.LoadAuth(service.GetWorkspaceID(), service.Name())
	if err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}

	clients, err := service.Clients()
	response := toggl.ClientsResponse{Clients: clients}
	defer func() {
		if err := service.SetParams(p.ServiceParams); err != nil {
			log.Printf("could not set service params: %v, reason: %v", p.ID, err)
			return
		}
		auth, err := svc.auth.LoadAuth(service.GetWorkspaceID(), service.Name())
		if err != nil {
			log.Printf("could not get service auth: %v, reason: %v", p.ID, err)
			return
		}
		if err := service.SetAuthData(auth.Data); err != nil {
			log.Printf("could not ser service auth data: %v, reason: %v", p.ID, err)
			return
		}

		workspaceID := service.GetWorkspaceID()
		objKey := service.KeyFor(toggl.ClientsPipeID)

		if err := svc.pipes.saveObject(workspaceID, objKey, response); err != nil {
			log.Printf("could not save object, workspaceID: %v key: %v, reason: %v", workspaceID, objKey, err)
			return
		}
	}()
	if err != nil {
		response.Error = err.Error()
		return err
	}
	connections, err := svc.conn.Load(service.GetWorkspaceID(), service.KeyFor(toggl.ClientsPipeID))
	if err != nil {
		response.Error = err.Error()
		return err
	}
	for _, client := range response.Clients {
		client.ID = connections.Data[client.ForeignID]
	}
	return nil
}

func (svc *Service) fetchProjects(p *Pipe) error {
	response := toggl.ProjectsResponse{}
	service := Create(p.ServiceID, p.WorkspaceID)

	defer func() {
		if err := service.SetParams(p.ServiceParams); err != nil {
			log.Printf("could not set service params: %v, reason: %v", p.ID, err)
			return
		}
		auth, err := svc.auth.LoadAuth(service.GetWorkspaceID(), service.Name())
		if err != nil {
			log.Printf("could not get service auth: %v, reason: %v", p.ID, err)
			return
		}
		if err := service.SetAuthData(auth.Data); err != nil {
			log.Printf("could not ser service auth data: %v, reason: %v", p.ID, err)
			return
		}

		workspaceID := service.GetWorkspaceID()
		objKey := service.KeyFor(toggl.ProjectsPipeID)

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

	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}

	auth, err := svc.auth.LoadAuth(service.GetWorkspaceID(), service.Name())
	if err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
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
	if clientConnections, err = svc.conn.Load(service.GetWorkspaceID(), service.KeyFor(toggl.ClientsPipeID)); err != nil {
		response.Error = err.Error()
		return err
	}
	if projectConnections, err = svc.conn.Load(service.GetWorkspaceID(), service.KeyFor(toggl.ProjectsPipeID)); err != nil {
		response.Error = err.Error()
		return err
	}

	for _, project := range response.Projects {
		project.ID = projectConnections.Data[project.ForeignID]
		project.ClientID = clientConnections.Data[project.ForeignClientID]
	}

	return nil
}

func (svc *Service) fetchTodoLists(p *Pipe) error {
	response := toggl.TasksResponse{}
	service := Create(p.ServiceID, p.WorkspaceID)

	defer func() {
		if err := service.SetParams(p.ServiceParams); err != nil {
			log.Printf("could not set service params: %v, reason: %v", p.ID, err)
			return
		}
		auth, err := svc.auth.LoadAuth(service.GetWorkspaceID(), service.Name())
		if err != nil {
			log.Printf("could not get service auth: %v, reason: %v", p.ID, err)
			return
		}
		if err := service.SetAuthData(auth.Data); err != nil {
			log.Printf("could not set auth data: %v, reason: %v", p.ID, err)
			return
		}

		workspaceID := service.GetWorkspaceID()
		objKey := service.KeyFor(toggl.TodoPipeID)

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

	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}
	auth, err := svc.auth.LoadAuth(service.GetWorkspaceID(), service.Name())
	if err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}

	service.SetSince(p.LastSync)
	tasks, err := service.TodoLists()
	if err != nil {
		response.Error = err.Error()
		return err
	}

	var projectConnections, taskConnections *connection.Connection

	if projectConnections, err = svc.conn.Load(service.GetWorkspaceID(), service.KeyFor(toggl.ProjectsPipeID)); err != nil {
		response.Error = err.Error()
		return err
	}
	if taskConnections, err = svc.conn.Load(service.GetWorkspaceID(), service.KeyFor(toggl.TodoPipeID)); err != nil {
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

func (svc *Service) fetchTasks(p *Pipe) error {
	response := toggl.TasksResponse{}
	service := Create(p.ServiceID, p.WorkspaceID)
	defer func() {
		if err := service.SetParams(p.ServiceParams); err != nil {
			log.Printf("could not set service params: %v, reason: %v", p.ID, err)
			return
		}

		auth, err := svc.auth.LoadAuth(service.GetWorkspaceID(), service.Name())
		if err != nil {
			log.Printf("could not get service auth: %v, reason: %v", p.ID, err)
			return
		}
		if err := service.SetAuthData(auth.Data); err != nil {
			log.Printf("could not set auth data: %v, reason: %v", p.ID, err)
			return
		}

		workspaceID := service.GetWorkspaceID()
		objKey := service.KeyFor(toggl.TasksPipeID)

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

	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}

	auth, err := svc.auth.LoadAuth(service.GetWorkspaceID(), service.Name())
	if err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}

	service.SetSince(p.LastSync)
	tasks, err := service.Tasks()
	if err != nil {
		response.Error = err.Error()
		return err
	}
	var projectConnections, taskConnections *connection.Connection

	if projectConnections, err = svc.conn.Load(service.GetWorkspaceID(), service.KeyFor(toggl.ProjectsPipeID)); err != nil {
		response.Error = err.Error()
		return err
	}
	if taskConnections, err = svc.conn.Load(service.GetWorkspaceID(), service.KeyFor(toggl.TasksPipeID)); err != nil {
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

func (svc *Service) fetchTimeEntries(p *Pipe) error {
	return nil
}

func (svc *Service) getClients(s ExternalService) (*toggl.ClientsResponse, error) {
	b, err := svc.pipes.getObject(s, toggl.ClientsPipeID)
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

func (svc *Service) getProjects(s ExternalService) (*toggl.ProjectsResponse, error) {
	b, err := svc.pipes.getObject(s, toggl.ProjectsPipeID)
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

func (svc *Service) getTasks(s ExternalService, objType string) (*toggl.TasksResponse, error) {
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

func (svc *Service) endSync(p *Pipe, saveStatus bool, err error) error {
	if !saveStatus {
		return err
	}

	if err != nil {
		// If it is JSON marshalling error suppress it for status
		if _, ok := err.(*json.UnmarshalTypeError); ok {
			err = ErrJSONParsing
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

func (svc *Service) postObjects(p *Pipe, saveStatus bool) (err error) {
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
		service := Create(p.ServiceID, p.WorkspaceID)
		if err := service.SetParams(p.ServiceParams); err != nil {
			log.Printf("could not set service params: %v, reason: %v", p.ID, err)
			break
		}
		auth, err := svc.auth.LoadAuth(service.GetWorkspaceID(), service.Name())
		if err != nil {
			log.Printf("could not get service auth: %v, reason: %v", p.ID, err)
			break
		}
		if err := service.SetAuthData(auth.Data); err != nil {
			log.Printf("could not set auth data: %v, reason: %v", p.ID, err)
			break
		}
		err = svc.postTimeEntries(p, service)
	default:
		panic(fmt.Sprintf("postObjects: Unrecognized pipeID - %s", p.ID))
	}
	return svc.endSync(p, saveStatus, err)
}

func (svc *Service) newStatus(p *Pipe) error {
	svc.pipes.loadLastSync(p)
	p.PipeStatus = NewPipeStatus(p.WorkspaceID, p.ServiceID, p.ID, svc.getPipesApiHost())
	return svc.pipes.savePipeStatus(p.PipeStatus)
}

func (svc *Service) getPipesApiHost() string {
	svc.mx.RLock()
	defer svc.mx.RUnlock()
	return svc.pipesApiHost
}

func (svc *Service) fillAvailablePipeTypes() *Service {
	svc.mx.Lock()
	defer svc.mx.Unlock()
	svc.availablePipeType = regexp.MustCompile("users|projects|todolists|todos|tasks|timeentries")
	return svc
}
