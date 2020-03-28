package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bugsnag/bugsnag-go"

	"github.com/toggl/pipes-api/pkg/integration"
	"github.com/toggl/pipes-api/pkg/toggl"
)

// ErrJSONParsing hides json marshalling errors from users
var ErrJSONParsing = errors.New("failed to parse response from service, please contact support")

type PipeFactory struct {
	*AuthorizationFactory
	AuthorizationsStorage
	PipesStorage
	ImportsStorage
	IDMappingsStorage
	TogglClient
}

func (pf *PipeFactory) Create(workspaceID int, sid integration.ID, pid integration.PipeID) *Pipe {
	return &Pipe{
		ID:          pid,
		Key:         PipesKey(sid, pid),
		ServiceID:   sid,
		WorkspaceID: workspaceID,

		AuthorizationFactory:  pf.AuthorizationFactory,
		AuthorizationsStorage: pf.AuthorizationsStorage,
		PipesStorage:          pf.PipesStorage,
		ImportsStorage:        pf.ImportsStorage,
		IDMappingsStorage:     pf.IDMappingsStorage,
		TogglClient:           pf.TogglClient,
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
	Key             string             `json:"-"`
	UsersSelector   []byte             `json:"-"`
	LastSync        *time.Time         `json:"-"`

	*AuthorizationFactory `json:"-"`
	AuthorizationsStorage `json:"-"`
	PipesStorage          `json:"-"`
	ImportsStorage        `json:"-"`
	IDMappingsStorage     `json:"-"`
	TogglClient           `json:"-"`

	pipesApiHost string       `json:"-"`
	mx           sync.RWMutex `json:"-"`
}

func PipesKey(sid integration.ID, pid integration.PipeID) string {
	return fmt.Sprintf("%s:%s", sid, pid)
}

func GetSidPidFromKey(key string) (integration.ID, integration.PipeID) {
	ids := strings.Split(key, ":")
	return integration.ID(ids[0]), integration.PipeID(ids[1])
}

func (p *Pipe) Synchronize() {
	var err error
	defer func() {
		if err != nil {
			// If it is JSON marshalling error suppress it for status
			if _, ok := err.(*json.UnmarshalTypeError); ok {
				err = ErrJSONParsing
			}
			p.PipeStatus.AddError(err)
		}
		if e := p.PipesStorage.SaveStatus(p.PipeStatus); e != nil {
			p.notifyBugsnag(e)
			log.Println(e)
		}
	}()

	p.mx.RLock()
	host := p.pipesApiHost
	p.mx.RUnlock()

	p.PipesStorage.LoadLastSyncFor(p)

	p.PipeStatus = NewPipeStatus(p.WorkspaceID, p.ServiceID, p.ID, host)
	if err = p.PipesStorage.SaveStatus(p.PipeStatus); err != nil {
		p.notifyBugsnag(err)
		return
	}

	auth := p.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
	if err := p.AuthorizationsStorage.Load(p.WorkspaceID, p.ServiceID, auth); err != nil {
		p.notifyBugsnag(err)
		return
	}
	if err = auth.Refresh(); err != nil {
		p.notifyBugsnag(err)
		return
	}
	p.TogglClient.WithAuthToken(auth.WorkspaceToken)

	switch p.ID {
	case integration.UsersPipe:
		p.syncUsers()
	case integration.ProjectsPipe:
		p.syncProjects()
	case integration.TodoListsPipe:
		p.syncTodoLists()
	case integration.TodosPipe, integration.TasksPipe:
		p.syncTasks()
	case integration.TimeEntriesPipe:
		p.syncTEs()
	default:
		bugsnag.Notify(fmt.Errorf("unrecognized pipeID: %s", p.ID))
		return
	}
}

func (p *Pipe) syncUsers() {
	err := p.FetchUsers()
	if err != nil {
		p.notifyBugsnag(err)
		return
	}

	err = p.postUsers()
	if err != nil {
		p.notifyBugsnag(err)
		return
	}
}

func (p *Pipe) syncProjects() {
	err := p.fetchProjects()
	if err != nil {
		p.notifyBugsnag(err)
		return
	}

	err = p.postProjects()
	if err != nil {
		p.notifyBugsnag(err)
		return
	}
}

func (p *Pipe) syncTodoLists() {
	err := p.fetchTodoLists()
	if err != nil {
		p.notifyBugsnag(err)
		return
	}

	err = p.postTodoLists()
	if err != nil {
		p.notifyBugsnag(err)
		return
	}
}

func (p *Pipe) syncTasks() {
	err := p.fetchTasks()
	if err != nil {
		p.notifyBugsnag(err)
		return
	}
	err = p.postTasks()
	if err != nil {
		p.notifyBugsnag(err)
		return
	}
}

func (p *Pipe) syncTEs() {
	service := NewExternalService(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		log.Printf("could not set service params: %v, reason: %v", p.ID, err)
		return
	}

	auth := p.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
	if err := p.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
		log.Printf("could not get service auth: %v, reason: %v", p.ID, err)
		return
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		log.Printf("could not set auth data: %v, reason: %v", p.ID, err)
		return
	}

	if err := p.postTimeEntries(service); err != nil {
		p.notifyBugsnag(err)
		return
	}
}

func (p *Pipe) postUsers() error {
	service := NewExternalService(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}

	auth := p.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
	if err := p.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
		return err
	}
	if err := auth.Refresh(); err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}

	usersResponse, err := p.ImportsStorage.LoadUsersFor(service)
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
	if err := json.Unmarshal(p.UsersSelector, &selector); err != nil {
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

	usersImport, err := p.TogglClient.PostUsers(integration.UsersPipe, toggl.UsersRequest{Users: users})
	if err != nil {
		return err
	}

	idMapping, err := p.IDMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(integration.UsersPipe))
	if err != nil {
		return err
	}
	for _, user := range usersImport.WorkspaceUsers {
		idMapping.Data[user.ForeignID] = user.ID
	}
	if err := p.IDMappingsStorage.Save(idMapping); err != nil {
		return err
	}

	p.PipeStatus.Complete(integration.UsersPipe, usersImport.Notifications, usersImport.Count())
	return nil
}

func (p *Pipe) postClients() error {
	service := NewExternalService(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}

	auth := p.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
	if err := p.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}

	clientsResponse, err := p.ImportsStorage.LoadClientsFor(service)
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
	clientsImport, err := p.TogglClient.PostClients(integration.ClientsPipe, clients)
	if err != nil {
		return err
	}
	var idMapping *IDMapping
	if idMapping, err = p.IDMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(integration.ClientsPipe)); err != nil {
		return err
	}
	for _, client := range clientsImport.Clients {
		idMapping.Data[client.ForeignID] = client.ID
	}
	if err := p.IDMappingsStorage.Save(idMapping); err != nil {
		return err
	}
	p.PipeStatus.Complete(integration.ClientsPipe, clientsImport.Notifications, clientsImport.Count())
	return nil
}

func (p *Pipe) postProjects() error {
	service := NewExternalService(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}

	auth := p.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
	if err := p.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
		return err
	}
	if err := auth.Refresh(); err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}

	projectsResponse, err := p.ImportsStorage.LoadProjectsFor(service)
	if err != nil {
		return errors.New("unable to get projects from DB")
	}
	if projectsResponse == nil {
		return errors.New("service projects not found")
	}
	projects := toggl.ProjectRequest{
		Projects: projectsResponse.Projects,
	}
	projectsImport, err := p.TogglClient.PostProjects(integration.ProjectsPipe, projects)
	if err != nil {
		return err
	}
	var idMapping *IDMapping
	if idMapping, err = p.IDMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(integration.ProjectsPipe)); err != nil {
		return err
	}
	for _, project := range projectsImport.Projects {
		idMapping.Data[project.ForeignID] = project.ID
	}
	if err := p.IDMappingsStorage.Save(idMapping); err != nil {
		return err
	}
	p.PipeStatus.Complete(integration.ProjectsPipe, projectsImport.Notifications, projectsImport.Count())
	return nil
}

func (p *Pipe) postTodoLists() error {
	service := NewExternalService(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}

	auth := p.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
	if err := p.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
		return err
	}
	if err := auth.Refresh(); err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}

	tasksResponse, err := p.ImportsStorage.LoadTodoListsFor(service)
	if err != nil {
		return errors.New("unable to get tasks from DB")
	}
	if tasksResponse == nil {
		return errors.New("service tasks not found")
	}
	trs, err := p.TogglClient.AdjustRequestSize(tasksResponse.Tasks, 1)
	if err != nil {
		return err
	}
	var notifications []string
	var count int
	for _, tr := range trs {
		tasksImport, err := p.TogglClient.PostTodoLists(integration.TasksPipe, tr) // TODO: WTF?? Why toggl.TasksPipe
		if err != nil {
			return err
		}
		idMapping, err := p.IDMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(integration.TodoListsPipe))
		if err != nil {
			return err
		}
		for _, task := range tasksImport.Tasks {
			idMapping.Data[task.ForeignID] = task.ID
		}
		if err := p.IDMappingsStorage.Save(idMapping); err != nil {
			return err
		}
		notifications = append(notifications, tasksImport.Notifications...)
		count += tasksImport.Count()
	}
	p.PipeStatus.Complete(integration.TodoListsPipe, notifications, count)
	return nil
}

func (p *Pipe) postTasks() error {
	service := NewExternalService(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}
	auth := p.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
	if err := p.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
		return err
	}
	if err := auth.Refresh(); err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}

	tasksResponse, err := p.ImportsStorage.LoadTasksFor(service)
	if err != nil {
		return errors.New("unable to get tasks from DB")
	}
	if tasksResponse == nil {
		return errors.New("service tasks not found")
	}
	trs, err := p.TogglClient.AdjustRequestSize(tasksResponse.Tasks, 1)
	if err != nil {
		return err
	}
	var notifications []string
	var count int
	for _, tr := range trs {
		tasksImport, err := p.TogglClient.PostTasks(integration.TasksPipe, tr)
		if err != nil {
			return err
		}
		idMapping, err := p.IDMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(integration.TasksPipe))
		if err != nil {
			return err
		}
		for _, task := range tasksImport.Tasks {
			idMapping.Data[task.ForeignID] = task.ID
		}
		if err := p.IDMappingsStorage.Save(idMapping); err != nil {
			return err
		}
		notifications = append(notifications, tasksImport.Notifications...)
		count += tasksImport.Count()
	}
	p.PipeStatus.Complete(p.ID, notifications, count)
	return nil
}

func (p *Pipe) postTimeEntries(service integration.Integration) error {
	usersIDMapping, err := p.IDMappingsStorage.LoadReversed(service.GetWorkspaceID(), service.KeyFor(integration.UsersPipe))
	if err != nil {
		return err
	}

	tasksIDMapping, err := p.IDMappingsStorage.LoadReversed(service.GetWorkspaceID(), service.KeyFor(integration.TasksPipe))
	if err != nil {
		return err
	}

	projectsIDMapping, err := p.IDMappingsStorage.LoadReversed(service.GetWorkspaceID(), service.KeyFor(integration.ProjectsPipe))
	if err != nil {
		return err
	}

	entriesIDMapping, err := p.IDMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(integration.TimeEntriesPipe))
	if err != nil {
		return err
	}

	if p.LastSync == nil {
		currentTime := time.Now()
		p.LastSync = &currentTime
	}

	timeEntries, err := p.TogglClient.GetTimeEntries(*p.LastSync, usersIDMapping.GetKeys(), projectsIDMapping.GetKeys())
	if err != nil {
		return err
	}

	for _, entry := range timeEntries {
		entry.ForeignID = strconv.Itoa(entriesIDMapping.Data[strconv.Itoa(entry.ID)])
		entry.ForeignTaskID = strconv.Itoa(tasksIDMapping.GetForeignID(entry.TaskID))
		entry.ForeignUserID = strconv.Itoa(usersIDMapping.GetForeignID(entry.UserID))
		entry.ForeignProjectID = strconv.Itoa(projectsIDMapping.GetForeignID(entry.ProjectID))

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
			entriesIDMapping.Data[strconv.Itoa(entry.ID)] = entryID
		}
	}

	if err := p.IDMappingsStorage.Save(entriesIDMapping); err != nil {
		return err
	}
	p.PipeStatus.Complete("timeentries", []string{}, len(timeEntries))
	return nil
}

func (p *Pipe) FetchUsers() error {
	service := NewExternalService(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}

	auth := p.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
	if err := p.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
		return err
	}

	if err := auth.Refresh(); err != nil {
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
		auth := p.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
		if err := p.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
			log.Printf("could not get service auth: %v, reason: %v", p.ID, err)
			return
		}
		if err := service.SetAuthData(auth.Data); err != nil {
			log.Printf("could not set auth data: %v, reason: %v", p.ID, err)
			return
		}

		if err := p.ImportsStorage.SaveUsersFor(service, response); err != nil {
			log.Printf("could not save object, workspaceID: %v key: %v, reason: %v", service.GetWorkspaceID(), service.KeyFor(integration.UsersPipe), err)
			return
		}
	}()
	if err != nil {
		response.Error = err.Error()
		return err
	}
	return nil
}

func (p *Pipe) fetchClients() error {
	service := NewExternalService(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}
	auth := p.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
	if err := p.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
		return err
	}
	if err := auth.Refresh(); err != nil {
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
		auth := p.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
		if err := p.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
			log.Printf("could not get service auth: %v, reason: %v", p.ID, err)
			return
		}
		if err := service.SetAuthData(auth.Data); err != nil {
			log.Printf("could not ser service auth data: %v, reason: %v", p.ID, err)
			return
		}

		if err := p.ImportsStorage.SaveClientsFor(service, response); err != nil {
			log.Printf("could not save object, workspaceID: %v key: %v, reason: %v", service.GetWorkspaceID(), service.KeyFor(integration.ClientsPipe), err)
			return
		}
	}()
	if err != nil {
		response.Error = err.Error()
		return err
	}
	clientsIDMapping, err := p.IDMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(integration.ClientsPipe))
	if err != nil {
		response.Error = err.Error()
		return err
	}
	for _, client := range response.Clients {
		client.ID = clientsIDMapping.Data[client.ForeignID]
	}
	return nil
}

func (p *Pipe) fetchProjects() error {
	response := toggl.ProjectsResponse{}
	service := NewExternalService(p.ServiceID, p.WorkspaceID)

	defer func() {
		if err := service.SetParams(p.ServiceParams); err != nil {
			log.Printf("could not set service params: %v, reason: %v", p.ID, err)
			return
		}
		auth := p.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
		if err := p.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
			log.Printf("could not get service auth: %v, reason: %v", p.ID, err)
			return
		}
		if err := service.SetAuthData(auth.Data); err != nil {
			log.Printf("could not ser service auth data: %v, reason: %v", p.ID, err)
			return
		}

		if err := p.ImportsStorage.SaveProjectsFor(service, response); err != nil {
			log.Printf("could not save object, workspaceID: %v key: %v, reason: %v", service.GetWorkspaceID(), service.KeyFor(integration.ProjectsPipe), err)
			return
		}
	}()

	if err := p.fetchClients(); err != nil {
		response.Error = err.Error()
		return err
	}
	if err := p.postClients(); err != nil {
		response.Error = err.Error()
		return err
	}

	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}

	auth := p.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
	if err := p.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
		return err
	}
	if err := auth.Refresh(); err != nil {
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

	var clientsIDMapping, projectsIDMapping *IDMapping
	if clientsIDMapping, err = p.IDMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(integration.ClientsPipe)); err != nil {
		response.Error = err.Error()
		return err
	}
	if projectsIDMapping, err = p.IDMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(integration.ProjectsPipe)); err != nil {
		response.Error = err.Error()
		return err
	}

	for _, project := range response.Projects {
		project.ID = projectsIDMapping.Data[project.ForeignID]
		project.ClientID = clientsIDMapping.Data[project.ForeignClientID]
	}

	return nil
}

func (p *Pipe) fetchTodoLists() error {
	response := toggl.TasksResponse{}
	service := NewExternalService(p.ServiceID, p.WorkspaceID)

	defer func() {
		if err := service.SetParams(p.ServiceParams); err != nil {
			log.Printf("could not set service params: %v, reason: %v", p.ID, err)
			return
		}
		auth := p.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
		if err := p.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
			log.Printf("could not get service auth: %v, reason: %v", p.ID, err)
			return
		}
		if err := service.SetAuthData(auth.Data); err != nil {
			log.Printf("could not set auth data: %v, reason: %v", p.ID, err)
			return
		}

		if err := p.ImportsStorage.SaveTodoListsFor(service, response); err != nil {
			log.Printf("could not save object, workspaceID: %v key: %v, reason: %v", service.GetWorkspaceID(), service.KeyFor(integration.TodoListsPipe), err)
			return
		}
	}()

	if err := p.fetchProjects(); err != nil {
		response.Error = err.Error()
		return err
	}
	if err := p.postProjects(); err != nil {
		response.Error = err.Error()
		return err
	}

	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}
	auth := p.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
	if err := p.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
		return err
	}
	if err := auth.Refresh(); err != nil {
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

	var projectsIDMapping, taskIDMapping *IDMapping

	if projectsIDMapping, err = p.IDMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(integration.ProjectsPipe)); err != nil {
		response.Error = err.Error()
		return err
	}
	if taskIDMapping, err = p.IDMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(integration.TodoListsPipe)); err != nil {
		response.Error = err.Error()
		return err
	}

	response.Tasks = make([]*toggl.Task, 0)
	for _, task := range tasks {
		id := taskIDMapping.Data[task.ForeignID]
		if (id > 0) || task.Active {
			task.ID = id
			task.ProjectID = projectsIDMapping.Data[task.ForeignProjectID]
			response.Tasks = append(response.Tasks, task)
		}
	}
	return nil
}

func (p *Pipe) fetchTasks() error {
	response := toggl.TasksResponse{}
	service := NewExternalService(p.ServiceID, p.WorkspaceID)
	defer func() {
		if err := service.SetParams(p.ServiceParams); err != nil {
			log.Printf("could not set service params: %v, reason: %v", p.ID, err)
			return
		}

		auth := p.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
		if err := p.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
			log.Printf("could not get service auth: %v, reason: %v", p.ID, err)
			return
		}
		if err := service.SetAuthData(auth.Data); err != nil {
			log.Printf("could not set auth data: %v, reason: %v", p.ID, err)
			return
		}

		if err := p.ImportsStorage.SaveTasksFor(service, response); err != nil {
			log.Printf("could not save object, workspaceID: %v key: %v, reason: %v", service.GetWorkspaceID(), service.KeyFor(integration.TasksPipe), err)
			return
		}
	}()

	if err := p.fetchProjects(); err != nil {
		response.Error = err.Error()
		return err
	}
	if err := p.postProjects(); err != nil {
		response.Error = err.Error()
		return err
	}

	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}

	auth := p.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
	if err := p.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
		return err
	}
	if err := auth.Refresh(); err != nil {
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
	var projectsIDMapping, taskIDMapping *IDMapping

	if projectsIDMapping, err = p.IDMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(integration.ProjectsPipe)); err != nil {
		response.Error = err.Error()
		return err
	}
	if taskIDMapping, err = p.IDMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(integration.TasksPipe)); err != nil {
		response.Error = err.Error()
		return err
	}

	response.Tasks = make([]*toggl.Task, 0)
	for _, task := range tasks {
		id := taskIDMapping.Data[task.ForeignID]
		if (id > 0) || task.Active {
			task.ID = id
			task.ProjectID = projectsIDMapping.Data[task.ForeignProjectID]
			response.Tasks = append(response.Tasks, task)
		}
	}
	return nil
}

func (p *Pipe) notifyBugsnag(err error) {
	meta := bugsnag.MetaData{
		"pipe": {
			"ID":            p.ID,
			"Name":          p.Name,
			"ServiceParams": string(p.ServiceParams),
			"WorkspaceID":   p.WorkspaceID,
			"ServiceID":     p.ServiceID,
		},
	}
	log.Println(err, meta)
	bugsnag.Notify(err, meta)
}
