package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	goauth2 "code.google.com/p/goauth2/oauth"
	"github.com/bugsnag/bugsnag-go"
	"github.com/tambet/oauthplain"

	"github.com/toggl/pipes-api/pkg/integrations"
	"github.com/toggl/pipes-api/pkg/oauth"
	"github.com/toggl/pipes-api/pkg/pipe"
	"github.com/toggl/pipes-api/pkg/toggl"
)

// mutex to prevent multiple of postPipeRun on same workspace run at same time
var postPipeRunWorkspaceLock = map[int]*sync.Mutex{}
var postPipeRunLock sync.Mutex

type Service struct {
	oauth oauth.Provider
	toggl pipe.TogglClient
	store pipe.Storage
	queue pipe.Queue

	pipesApiHost string
	mx           sync.RWMutex
}

func NewService(oauth oauth.Provider, store pipe.Storage, queue pipe.Queue, toggl pipe.TogglClient, pipesApiHost string) *Service {

	svc := &Service{
		toggl: toggl,
		oauth: oauth,
		store: store,
		queue: queue,

		pipesApiHost: pipesApiHost,
	}

	return svc
}

func (svc *Service) GetPipe(workspaceID int, serviceID integrations.ExternalServiceID, pipeID integrations.PipeID) (*pipe.Pipe, error) {
	p, err := svc.store.LoadPipe(workspaceID, serviceID, pipeID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		p = pipe.NewPipe(workspaceID, serviceID, pipeID)
	}

	p.PipeStatus, err = svc.store.LoadPipeStatus(workspaceID, serviceID, pipeID)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (svc *Service) CreatePipe(workspaceID int, serviceID integrations.ExternalServiceID, pipeID integrations.PipeID, params []byte) error {
	p := pipe.NewPipe(workspaceID, serviceID, pipeID)

	service := pipe.NewExternalService(serviceID, workspaceID)
	err := service.SetParams(params)
	if err != nil {
		return SetParamsError{err}
	}
	p.ServiceParams = params

	if err := svc.store.Save(p); err != nil {
		return err
	}
	return nil
}

func (svc *Service) UpdatePipe(workspaceID int, serviceID integrations.ExternalServiceID, pipeID integrations.PipeID, params []byte) error {
	p, err := svc.store.LoadPipe(workspaceID, serviceID, pipeID)
	if err != nil {
		return err
	}
	if p == nil {
		return ErrPipeNotConfigured
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return err
	}
	if err := svc.store.Save(p); err != nil {
		return err
	}
	return nil
}

func (svc *Service) DeletePipe(workspaceID int, serviceID integrations.ExternalServiceID, pipeID integrations.PipeID) error {
	p, err := svc.store.LoadPipe(workspaceID, serviceID, pipeID)
	if err != nil {
		return err
	}
	if p == nil {
		return ErrPipeNotConfigured
	}
	if err := svc.store.Delete(p, workspaceID); err != nil {
		return err
	}
	return nil
}

var ErrNoContent = errors.New("no content")

func (svc *Service) GetServicePipeLog(workspaceID int, serviceID integrations.ExternalServiceID, pipeID integrations.PipeID) (string, error) {
	pipeStatus, err := svc.store.LoadPipeStatus(workspaceID, serviceID, pipeID)
	if err != nil {
		return "", err
	}
	if pipeStatus == nil {
		return "", ErrNoContent
	}
	return pipeStatus.GenerateLog(), nil
}

// TODO: Remove (Probably dead method).
func (svc *Service) ClearIDMappings(workspaceID int, serviceID integrations.ExternalServiceID, pipeID integrations.PipeID) error {
	p, err := svc.store.LoadPipe(workspaceID, serviceID, pipeID)
	if err != nil {
		return err
	}
	if p == nil {
		return ErrPipeNotConfigured
	}
	service := pipe.NewExternalService(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}
	auth, err := svc.store.LoadAuthorization(service.GetWorkspaceID(), service.ID())
	if err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}
	pipeStatus, err := svc.store.LoadPipeStatus(p.WorkspaceID, p.ServiceID, p.ID)
	if err != nil {
		return err
	}

	err = svc.store.DeleteIDMappings(p.WorkspaceID, service.KeyFor(p.ID), pipeStatus.Key)
	if err != nil {
		return err
	}
	return nil
}

func (svc *Service) RunPipe(workspaceID int, serviceID integrations.ExternalServiceID, pipeID integrations.PipeID, usersSelector []byte) error {
	// make sure no race condition on fetching workspace lock
	postPipeRunLock.Lock()
	wsLock, exists := postPipeRunWorkspaceLock[workspaceID]
	if !exists {
		wsLock = &sync.Mutex{}
		postPipeRunWorkspaceLock[workspaceID] = wsLock
	}
	postPipeRunLock.Unlock()

	p, err := svc.store.LoadPipe(workspaceID, serviceID, pipeID)
	if err != nil {
		return err
	}
	if p == nil {
		return ErrPipeNotConfigured
	}

	if p.ID == integrations.UsersPipe {
		p.UsersSelector = usersSelector
		if len(p.UsersSelector) == 0 {
			return SetParamsError{errors.New("Missing request payload")}
		}

		go func() {
			wsLock.Lock()
			svc.Run(p)
			wsLock.Unlock()
		}()
		time.Sleep(500 * time.Millisecond) // TODO: Is that synchronization ? :D
		return nil
	}

	if err := svc.queue.QueuePipeAsFirst(p); err != nil {
		return err
	}
	return nil
}

func (svc *Service) GetServiceUsers(workspaceID int, serviceID integrations.ExternalServiceID, forceImport bool) (*toggl.UsersResponse, error) {
	service := pipe.NewExternalService(serviceID, workspaceID)
	auth, err := svc.store.LoadAuthorization(service.GetWorkspaceID(), service.ID())
	if err != nil {
		return nil, LoadError{err}
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return nil, err
	}

	usersPipe, err := svc.store.LoadPipe(workspaceID, serviceID, integrations.UsersPipe)
	if err != nil {
		return nil, err
	}
	if usersPipe == nil {
		return nil, ErrPipeNotConfigured
	}
	if err := service.SetParams(usersPipe.ServiceParams); err != nil {
		return nil, SetParamsError{err}
	}

	if forceImport {
		if err := svc.store.DeleteUsersFor(service); err != nil {
			return nil, err
		}
	}

	usersResponse, err := svc.store.LoadUsersFor(service)
	if err != nil {
		return nil, err
	}

	if usersResponse == nil {
		if forceImport {
			go func() {
				fetchErr := svc.fetchUsers(usersPipe)
				if fetchErr != nil {
					log.Print(fetchErr.Error())
				}
			}()
		}
		return nil, ErrNoContent
	}
	return usersResponse, nil
}

func (svc *Service) GetServiceAccounts(workspaceID int, serviceID integrations.ExternalServiceID, forceImport bool) (*toggl.AccountsResponse, error) {
	service := pipe.NewExternalService(serviceID, workspaceID)
	auth, err := svc.store.LoadAuthorization(service.GetWorkspaceID(), service.ID())
	if err != nil {
		return nil, LoadError{err}
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return nil, err
	}

	if err := svc.refreshAuthorization(auth); err != nil {
		return nil, RefreshError{errors.New("oAuth refresh failed!")}
	}
	if forceImport {
		if err := svc.store.DeleteAccountsFor(service); err != nil {
			return nil, err
		}
	}

	accountsResponse, err := svc.store.LoadAccountsFor(service)
	if err != nil {
		return nil, err
	}

	if accountsResponse == nil {
		go func() {
			var response toggl.AccountsResponse
			accounts, err := service.Accounts()
			response.Accounts = accounts
			if err != nil {
				response.Error = err.Error()
			}
			if err := svc.store.SaveAccountsFor(service, response); err != nil {
				log.Print(err.Error())
			}
		}()
		return nil, ErrNoContent
	}
	return accountsResponse, nil
}

func (svc *Service) GetAuthURL(serviceID integrations.ExternalServiceID, accountName, callbackURL string) (string, error) {
	config, found := svc.oauth.OAuth1Configs(serviceID)
	if !found {
		return "", LoadError{errors.New("env OAuth config not found")}
	}
	transport := &oauthplain.Transport{
		Config: config.UpdateURLs(accountName),
	}
	token, err := transport.AuthCodeURL(callbackURL)
	if err != nil {
		return "", err
	}
	return token.AuthorizeUrl, nil
}

func (svc *Service) CreateAuthorization(workspaceID int, serviceID integrations.ExternalServiceID, workspaceToken string, params pipe.AuthParams) error {
	auth := pipe.NewAuthorization(workspaceID, serviceID, workspaceToken)
	authType, err := svc.store.LoadAuthorizationType(serviceID)
	if err != nil {
		return err
	}
	switch authType {
	case pipe.TypeOauth1:
		token, err := svc.oauth.OAuth1Exchange(serviceID, params.AccountName, params.Token, params.Verifier)
		if err != nil {
			return err
		}
		if err := auth.SetOAuth1Token(token); err != nil {
			return err
		}
	case pipe.TypeOauth2:
		if params.Code == "" {
			return errors.New("missing code")
		}
		token, err := svc.oauth.OAuth2Exchange(serviceID, params.Code)
		if err != nil {
			return err
		}
		if err := auth.SetOAuth2Token(token); err != nil {
			return err
		}
	}

	if err := svc.store.SaveAuthorization(auth); err != nil {
		return err
	}
	return nil
}

func (svc *Service) DeleteAuthorization(workspaceID int, serviceID integrations.ExternalServiceID) error {
	if err := svc.store.DeleteAuthorization(workspaceID, serviceID); err != nil {
		return err
	}
	if err := svc.store.DeletePipesByWorkspaceIDServiceID(workspaceID, serviceID); err != nil {
		return err
	}
	return nil
}

func (svc *Service) GetIntegrations(workspaceID int) ([]pipe.Integration, error) {
	authorizations, err := svc.store.LoadWorkspaceAuthorizations(workspaceID)
	if err != nil {
		return nil, err
	}
	workspacePipes, err := svc.store.LoadPipes(workspaceID)
	if err != nil {
		return nil, err
	}
	pipeStatuses, err := svc.store.LoadPipeStatuses(workspaceID)
	if err != nil {
		return nil, err
	}

	allIntegrations, err := svc.store.LoadIntegrations()
	if err != nil {
		return nil, err
	}

	var igr []pipe.Integration
	for _, current := range allIntegrations {
		var integration = current
		integration.AuthURL = svc.oauth.OAuth2URL(integration.ID)
		integration.Authorized = authorizations[integration.ID]
		var pipes []*pipe.Pipe
		for i := range integration.Pipes {
			var p = *integration.Pipes[i]
			key := pipe.PipesKey(integration.ID, p.ID)
			existingPipe := workspacePipes[key]
			if existingPipe != nil {
				p.Automatic = existingPipe.Automatic
				p.Configured = existingPipe.Configured
			}

			p.PipeStatus = pipeStatuses[key]
			pipes = append(pipes, &p)
		}
		integration.Pipes = pipes
		igr = append(igr, *integration)
	}
	return igr, nil
}

func (svc *Service) Ready() []error {
	errs := make([]error, 0)

	if svc.store.IsDown() {
		errs = append(errs, errors.New("database is down"))
	}

	if err := svc.toggl.Ping(); err != nil {
		errs = append(errs, err)
	}
	return errs
}

func (svc *Service) Run(p *pipe.Pipe) {
	var err error
	defer func() {
		if err != nil {
			// If it is JSON marshalling error suppress it for status
			if _, ok := err.(*json.UnmarshalTypeError); ok {
				err = ErrJSONParsing
			}
			p.PipeStatus.AddError(err)
		}
		if e := svc.store.SavePipeStatus(p.PipeStatus); e != nil {
			svc.notifyBugsnag(e, p)
			log.Println(e)
		}
	}()

	svc.mx.RLock()
	host := svc.pipesApiHost
	svc.mx.RUnlock()

	svc.store.LoadLastSync(p)

	p.PipeStatus = pipe.NewPipeStatus(p.WorkspaceID, p.ServiceID, p.ID, host)
	if err = svc.store.SavePipeStatus(p.PipeStatus); err != nil {
		svc.notifyBugsnag(err, p)
		return
	}

	auth, err := svc.store.LoadAuthorization(p.WorkspaceID, p.ServiceID)
	if err != nil {
		svc.notifyBugsnag(err, p)
		return
	}
	if err = svc.refreshAuthorization(auth); err != nil {
		svc.notifyBugsnag(err, p)
		return
	}
	svc.toggl.WithAuthToken(auth.WorkspaceToken)

	switch p.ID {
	case integrations.UsersPipe:
		svc.syncUsers(p)
	case integrations.ProjectsPipe:
		svc.syncProjects(p)
	case integrations.TodoListsPipe:
		svc.syncTodoLists(p)
	case integrations.TodosPipe, integrations.TasksPipe:
		svc.syncTasks(p)
	case integrations.TimeEntriesPipe:
		svc.syncTEs(p)
	default:
		bugsnag.Notify(fmt.Errorf("unrecognized pipeID: %s", p.ID))
		return
	}
}

func (svc *Service) syncUsers(p *pipe.Pipe) {
	err := svc.fetchUsers(p)
	if err != nil {
		svc.notifyBugsnag(err, p)
		return
	}

	err = svc.postUsers(p)
	if err != nil {
		svc.notifyBugsnag(err, p)
		return
	}
}

func (svc *Service) syncProjects(p *pipe.Pipe) {
	err := svc.fetchProjects(p)
	if err != nil {
		svc.notifyBugsnag(err, p)
		return
	}

	err = svc.postProjects(p)
	if err != nil {
		svc.notifyBugsnag(err, p)
		return
	}
}

func (svc *Service) syncTodoLists(p *pipe.Pipe) {
	err := svc.fetchTodoLists(p)
	if err != nil {
		svc.notifyBugsnag(err, p)
		return
	}

	err = svc.postTodoLists(p)
	if err != nil {
		svc.notifyBugsnag(err, p)
		return
	}
}

func (svc *Service) syncTasks(p *pipe.Pipe) {
	err := svc.fetchTasks(p)
	if err != nil {
		svc.notifyBugsnag(err, p)
		return
	}
	err = svc.postTasks(p)
	if err != nil {
		svc.notifyBugsnag(err, p)
		return
	}
}

func (svc *Service) syncTEs(p *pipe.Pipe) {
	service := pipe.NewExternalService(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		log.Printf("could not set service params: %v, reason: %v", p.ID, err)
		return
	}
	auth, err := svc.store.LoadAuthorization(service.GetWorkspaceID(), service.ID())
	if err != nil {
		log.Printf("could not get service auth: %v, reason: %v", p.ID, err)
		return
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		log.Printf("could not set auth data: %v, reason: %v", p.ID, err)
		return
	}

	err = svc.postTimeEntries(p, service)
	if err != nil {
		svc.notifyBugsnag(err, p)
		return
	}
}

func (svc *Service) refreshAuthorization(a *pipe.Authorization) error {
	authType, err := svc.store.LoadAuthorizationType(a.ServiceID)
	if err != nil {
		return err
	}
	if authType != pipe.TypeOauth2 {
		return nil
	}
	var token goauth2.Token
	if err := json.Unmarshal(a.Data, &token); err != nil {
		return err
	}
	if !token.Expired() {
		return nil
	}
	config, res := svc.oauth.OAuth2Configs(a.ServiceID)
	if !res {
		return errors.New("service OAuth config not found")
	}
	if err := svc.oauth.OAuth2Refresh(config, &token); err != nil {
		return err
	}
	if err := a.SetOAuth2Token(&token); err != nil {
		return err
	}
	if err := svc.store.SaveAuthorization(a); err != nil {
		return err
	}
	return nil
}

func (svc *Service) notifyBugsnag(err error, p *pipe.Pipe) {
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

func (svc *Service) postUsers(p *pipe.Pipe) error {
	service := pipe.NewExternalService(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}

	auth, err := svc.store.LoadAuthorization(service.GetWorkspaceID(), service.ID())
	if err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}

	usersResponse, err := svc.store.LoadUsersFor(service)
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

	usersImport, err := svc.toggl.PostUsers(integrations.UsersPipe, toggl.UsersRequest{Users: users})
	if err != nil {
		return err
	}

	idMapping, err := svc.store.LoadIDMapping(service.GetWorkspaceID(), service.KeyFor(integrations.UsersPipe))
	if err != nil {
		return err
	}
	for _, user := range usersImport.WorkspaceUsers {
		idMapping.Data[user.ForeignID] = user.ID
	}
	if err := svc.store.SaveIDMapping(idMapping); err != nil {
		return err
	}

	p.PipeStatus.Complete(integrations.UsersPipe, usersImport.Notifications, usersImport.Count())
	return nil
}

func (svc *Service) postClients(p *pipe.Pipe) error {
	service := pipe.NewExternalService(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}
	auth, err := svc.store.LoadAuthorization(service.GetWorkspaceID(), service.ID())
	if err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}

	clientsResponse, err := svc.store.LoadClientsFor(service)
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
	clientsImport, err := svc.toggl.PostClients(integrations.ClientsPipe, clients)
	if err != nil {
		return err
	}
	var idMapping *pipe.IDMapping
	if idMapping, err = svc.store.LoadIDMapping(service.GetWorkspaceID(), service.KeyFor(integrations.ClientsPipe)); err != nil {
		return err
	}
	for _, client := range clientsImport.Clients {
		idMapping.Data[client.ForeignID] = client.ID
	}
	if err := svc.store.SaveIDMapping(idMapping); err != nil {
		return err
	}
	p.PipeStatus.Complete(integrations.ClientsPipe, clientsImport.Notifications, clientsImport.Count())
	return nil
}

func (svc *Service) postProjects(p *pipe.Pipe) error {
	service := pipe.NewExternalService(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}

	auth, err := svc.store.LoadAuthorization(service.GetWorkspaceID(), service.ID())
	if err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}

	projectsResponse, err := svc.store.LoadProjectsFor(service)
	if err != nil {
		return errors.New("unable to get projects from DB")
	}
	if projectsResponse == nil {
		return errors.New("service projects not found")
	}
	projects := toggl.ProjectRequest{
		Projects: projectsResponse.Projects,
	}
	projectsImport, err := svc.toggl.PostProjects(integrations.ProjectsPipe, projects)
	if err != nil {
		return err
	}
	var idMapping *pipe.IDMapping
	if idMapping, err = svc.store.LoadIDMapping(service.GetWorkspaceID(), service.KeyFor(integrations.ProjectsPipe)); err != nil {
		return err
	}
	for _, project := range projectsImport.Projects {
		idMapping.Data[project.ForeignID] = project.ID
	}
	if err := svc.store.SaveIDMapping(idMapping); err != nil {
		return err
	}
	p.PipeStatus.Complete(integrations.ProjectsPipe, projectsImport.Notifications, projectsImport.Count())
	return nil
}

func (svc *Service) postTodoLists(p *pipe.Pipe) error {
	service := pipe.NewExternalService(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}

	auth, err := svc.store.LoadAuthorization(service.GetWorkspaceID(), service.ID())
	if err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}

	tasksResponse, err := svc.store.LoadTodoListsFor(service)
	if err != nil {
		return errors.New("unable to get tasks from DB")
	}
	if tasksResponse == nil {
		return errors.New("service tasks not found")
	}
	trs, err := svc.toggl.AdjustRequestSize(tasksResponse.Tasks, 1)
	if err != nil {
		return err
	}
	var notifications []string
	var count int
	for _, tr := range trs {
		tasksImport, err := svc.toggl.PostTodoLists(integrations.TasksPipe, tr) // TODO: WTF?? Why toggl.TasksPipe
		if err != nil {
			return err
		}
		idMapping, err := svc.store.LoadIDMapping(service.GetWorkspaceID(), service.KeyFor(integrations.TodoListsPipe))
		if err != nil {
			return err
		}
		for _, task := range tasksImport.Tasks {
			idMapping.Data[task.ForeignID] = task.ID
		}
		if err := svc.store.SaveIDMapping(idMapping); err != nil {
			return err
		}
		notifications = append(notifications, tasksImport.Notifications...)
		count += tasksImport.Count()
	}
	p.PipeStatus.Complete(integrations.TodoListsPipe, notifications, count)
	return nil
}

func (svc *Service) postTasks(p *pipe.Pipe) error {
	service := pipe.NewExternalService(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}
	auth, err := svc.store.LoadAuthorization(service.GetWorkspaceID(), service.ID())
	if err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}

	tasksResponse, err := svc.store.LoadTasksFor(service)
	if err != nil {
		return errors.New("unable to get tasks from DB")
	}
	if tasksResponse == nil {
		return errors.New("service tasks not found")
	}
	trs, err := svc.toggl.AdjustRequestSize(tasksResponse.Tasks, 1)
	if err != nil {
		return err
	}
	var notifications []string
	var count int
	for _, tr := range trs {
		tasksImport, err := svc.toggl.PostTasks(integrations.TasksPipe, tr)
		if err != nil {
			return err
		}
		idMapping, err := svc.store.LoadIDMapping(service.GetWorkspaceID(), service.KeyFor(integrations.TasksPipe))
		if err != nil {
			return err
		}
		for _, task := range tasksImport.Tasks {
			idMapping.Data[task.ForeignID] = task.ID
		}
		if err := svc.store.SaveIDMapping(idMapping); err != nil {
			return err
		}
		notifications = append(notifications, tasksImport.Notifications...)
		count += tasksImport.Count()
	}
	p.PipeStatus.Complete(p.ID, notifications, count)
	return nil
}

func (svc *Service) postTimeEntries(p *pipe.Pipe, service integrations.ExternalService) error {
	usersIDMapping, err := svc.store.LoadReversedIDMapping(service.GetWorkspaceID(), service.KeyFor(integrations.UsersPipe))
	if err != nil {
		return err
	}

	tasksIDMapping, err := svc.store.LoadReversedIDMapping(service.GetWorkspaceID(), service.KeyFor(integrations.TasksPipe))
	if err != nil {
		return err
	}

	projectsIDMapping, err := svc.store.LoadReversedIDMapping(service.GetWorkspaceID(), service.KeyFor(integrations.ProjectsPipe))
	if err != nil {
		return err
	}

	entriesIDMapping, err := svc.store.LoadIDMapping(service.GetWorkspaceID(), service.KeyFor(integrations.TimeEntriesPipe))
	if err != nil {
		return err
	}

	if p.LastSync == nil {
		currentTime := time.Now()
		p.LastSync = &currentTime
	}

	timeEntries, err := svc.toggl.GetTimeEntries(*p.LastSync, usersIDMapping.GetKeys(), projectsIDMapping.GetKeys())
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

	if err := svc.store.SaveIDMapping(entriesIDMapping); err != nil {
		return err
	}
	p.PipeStatus.Complete("timeentries", []string{}, len(timeEntries))
	return nil
}

func (svc *Service) fetchUsers(p *pipe.Pipe) error {
	service := pipe.NewExternalService(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}

	auth, err := svc.store.LoadAuthorization(service.GetWorkspaceID(), service.ID())
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
		auth, err := svc.store.LoadAuthorization(service.GetWorkspaceID(), service.ID())
		if err != nil {
			log.Printf("could not get service auth: %v, reason: %v", p.ID, err)
			return
		}
		if err := service.SetAuthData(auth.Data); err != nil {
			log.Printf("could not set auth data: %v, reason: %v", p.ID, err)
			return
		}

		if err := svc.store.SaveUsersFor(service, response); err != nil {
			log.Printf("could not save object, workspaceID: %v key: %v, reason: %v", service.GetWorkspaceID(), service.KeyFor(integrations.UsersPipe), err)
			return
		}
	}()
	if err != nil {
		response.Error = err.Error()
		return err
	}
	return nil
}

func (svc *Service) fetchClients(p *pipe.Pipe) error {
	service := pipe.NewExternalService(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}
	auth, err := svc.store.LoadAuthorization(service.GetWorkspaceID(), service.ID())
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
		auth, err := svc.store.LoadAuthorization(service.GetWorkspaceID(), service.ID())
		if err != nil {
			log.Printf("could not get service auth: %v, reason: %v", p.ID, err)
			return
		}
		if err := service.SetAuthData(auth.Data); err != nil {
			log.Printf("could not ser service auth data: %v, reason: %v", p.ID, err)
			return
		}

		if err := svc.store.SaveClientsFor(service, response); err != nil {
			log.Printf("could not save object, workspaceID: %v key: %v, reason: %v", service.GetWorkspaceID(), service.KeyFor(integrations.ClientsPipe), err)
			return
		}
	}()
	if err != nil {
		response.Error = err.Error()
		return err
	}
	clientsIDMapping, err := svc.store.LoadIDMapping(service.GetWorkspaceID(), service.KeyFor(integrations.ClientsPipe))
	if err != nil {
		response.Error = err.Error()
		return err
	}
	for _, client := range response.Clients {
		client.ID = clientsIDMapping.Data[client.ForeignID]
	}
	return nil
}

func (svc *Service) fetchProjects(p *pipe.Pipe) error {
	response := toggl.ProjectsResponse{}
	service := pipe.NewExternalService(p.ServiceID, p.WorkspaceID)

	defer func() {
		if err := service.SetParams(p.ServiceParams); err != nil {
			log.Printf("could not set service params: %v, reason: %v", p.ID, err)
			return
		}
		auth, err := svc.store.LoadAuthorization(service.GetWorkspaceID(), service.ID())
		if err != nil {
			log.Printf("could not get service auth: %v, reason: %v", p.ID, err)
			return
		}
		if err := service.SetAuthData(auth.Data); err != nil {
			log.Printf("could not ser service auth data: %v, reason: %v", p.ID, err)
			return
		}

		if err := svc.store.SaveProjectsFor(service, response); err != nil {
			log.Printf("could not save object, workspaceID: %v key: %v, reason: %v", service.GetWorkspaceID(), service.KeyFor(integrations.ProjectsPipe), err)
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

	auth, err := svc.store.LoadAuthorization(service.GetWorkspaceID(), service.ID())
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

	var clientsIDMapping, projectsIDMapping *pipe.IDMapping
	if clientsIDMapping, err = svc.store.LoadIDMapping(service.GetWorkspaceID(), service.KeyFor(integrations.ClientsPipe)); err != nil {
		response.Error = err.Error()
		return err
	}
	if projectsIDMapping, err = svc.store.LoadIDMapping(service.GetWorkspaceID(), service.KeyFor(integrations.ProjectsPipe)); err != nil {
		response.Error = err.Error()
		return err
	}

	for _, project := range response.Projects {
		project.ID = projectsIDMapping.Data[project.ForeignID]
		project.ClientID = clientsIDMapping.Data[project.ForeignClientID]
	}

	return nil
}

func (svc *Service) fetchTodoLists(p *pipe.Pipe) error {
	response := toggl.TasksResponse{}
	service := pipe.NewExternalService(p.ServiceID, p.WorkspaceID)

	defer func() {
		if err := service.SetParams(p.ServiceParams); err != nil {
			log.Printf("could not set service params: %v, reason: %v", p.ID, err)
			return
		}
		auth, err := svc.store.LoadAuthorization(service.GetWorkspaceID(), service.ID())
		if err != nil {
			log.Printf("could not get service auth: %v, reason: %v", p.ID, err)
			return
		}
		if err := service.SetAuthData(auth.Data); err != nil {
			log.Printf("could not set auth data: %v, reason: %v", p.ID, err)
			return
		}

		if err := svc.store.SaveTodoListsFor(service, response); err != nil {
			log.Printf("could not save object, workspaceID: %v key: %v, reason: %v", service.GetWorkspaceID(), service.KeyFor(integrations.TodoListsPipe), err)
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
	auth, err := svc.store.LoadAuthorization(service.GetWorkspaceID(), service.ID())
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

	var projectsIDMapping, taskIDMapping *pipe.IDMapping

	if projectsIDMapping, err = svc.store.LoadIDMapping(service.GetWorkspaceID(), service.KeyFor(integrations.ProjectsPipe)); err != nil {
		response.Error = err.Error()
		return err
	}
	if taskIDMapping, err = svc.store.LoadIDMapping(service.GetWorkspaceID(), service.KeyFor(integrations.TodoListsPipe)); err != nil {
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

func (svc *Service) fetchTasks(p *pipe.Pipe) error {
	response := toggl.TasksResponse{}
	service := pipe.NewExternalService(p.ServiceID, p.WorkspaceID)
	defer func() {
		if err := service.SetParams(p.ServiceParams); err != nil {
			log.Printf("could not set service params: %v, reason: %v", p.ID, err)
			return
		}

		auth, err := svc.store.LoadAuthorization(service.GetWorkspaceID(), service.ID())
		if err != nil {
			log.Printf("could not get service auth: %v, reason: %v", p.ID, err)
			return
		}
		if err := service.SetAuthData(auth.Data); err != nil {
			log.Printf("could not set auth data: %v, reason: %v", p.ID, err)
			return
		}

		if err := svc.store.SaveTasksFor(service, response); err != nil {
			log.Printf("could not save object, workspaceID: %v key: %v, reason: %v", service.GetWorkspaceID(), service.KeyFor(integrations.TasksPipe), err)
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

	auth, err := svc.store.LoadAuthorization(service.GetWorkspaceID(), service.ID())
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
	var projectsIDMapping, taskIDMapping *pipe.IDMapping

	if projectsIDMapping, err = svc.store.LoadIDMapping(service.GetWorkspaceID(), service.KeyFor(integrations.ProjectsPipe)); err != nil {
		response.Error = err.Error()
		return err
	}
	if taskIDMapping, err = svc.store.LoadIDMapping(service.GetWorkspaceID(), service.KeyFor(integrations.TasksPipe)); err != nil {
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
