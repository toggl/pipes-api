package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"strconv"
	"strings"
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

	availablePipeType     *regexp.Regexp
	availableServiceType  *regexp.Regexp
	availableIntegrations []*pipe.Integration
	// Stores available authorization types for each service
	// Map format: map[externalServiceID]authType
	availableAuthTypes map[integrations.ExternalServiceID]string
	pipesApiHost       string
	mx                 sync.RWMutex
}

func NewService(oauth oauth.Provider, store pipe.Storage, queue pipe.Queue, toggl pipe.TogglClient, pipesApiHost string) *Service {

	svc := &Service{
		toggl: toggl,
		oauth: oauth,
		store: store,
		queue: queue,

		pipesApiHost:          pipesApiHost,
		availableIntegrations: []*pipe.Integration{},
		availableAuthTypes:    map[integrations.ExternalServiceID]string{},
	}

	return svc
}

func (svc *Service) LoadIntegrationsFromConfig(integrationsConfigPath string) {
	svc.loadIntegrations(integrationsConfigPath).fillAvailableServices().fillAvailablePipeTypes()
	svc.mx.RLock()
	for _, integration := range svc.availableIntegrations {
		svc.availableAuthTypes[integration.ID] = integration.AuthType
	}
	svc.mx.RUnlock()
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
		if err := svc.store.ClearImportFor(service, integrations.UsersPipe); err != nil {
			return nil, err
		}
	}

	usersResponse, err := svc.getUsers(service)
	if err != nil {
		return nil, err
	}

	if usersResponse == nil {
		if forceImport {
			go func() {
				fetchErr := svc.fetchUsers(usersPipe)
				if fetchErr != nil {
					log.Print(err.Error())
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
		if err := svc.store.ClearImportFor(service, "accounts"); err != nil { // TODO: Why "accounts". We have no accounts store. ???
			return nil, err
		}
	}

	accountsResponse, err := svc.store.LoadAccounts(service)
	if err != nil {
		return nil, err
	}

	if accountsResponse == nil {
		go func() {
			if err := svc.store.SaveAccounts(service); err != nil {
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

// CreateAuthorization - creates new authorization for workspace and specified service.
// Parameter: currentWorkspaceToken is an "Toggl.Track" API token or UserName.
func (svc *Service) CreateAuthorization(workspaceID int, serviceID integrations.ExternalServiceID, currentWorkspaceToken string, oAuthRawData []byte) error {
	var err error
	auth := pipe.NewAuthorization(workspaceID, serviceID)
	auth.WorkspaceToken = currentWorkspaceToken

	switch svc.getAvailableAuthorizations(serviceID) {
	case pipe.TypeOauth1:
		var params oauth.ParamsV1
		err := json.Unmarshal(oAuthRawData, &params)
		if err != nil {
			return err
		}
		auth.Data, err = svc.oauth.OAuth1Exchange(serviceID, params)
	case pipe.TypeOauth2:
		var payload map[string]interface{}
		err := json.Unmarshal(oAuthRawData, &payload)
		if err != nil {
			return err
		}
		oAuth2Code := payload["code"].(string)
		if oAuth2Code == "" {
			return errors.New("missing code")
		}
		auth.Data, err = svc.oauth.OAuth2Exchange(serviceID, oAuth2Code)
	}
	if err != nil {
		return err
	}

	if err := svc.store.SaveAuthorization(auth); err != nil {
		return err
	}
	return nil
}

func (svc *Service) DeleteAuthorization(workspaceID int, serviceID integrations.ExternalServiceID) error {
	service := pipe.NewExternalService(serviceID, workspaceID)
	auth, err := svc.store.LoadAuthorization(service.GetWorkspaceID(), service.ID())
	if err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}

	if err := svc.store.DeleteAuthorization(service.GetWorkspaceID(), service.ID()); err != nil {
		return err
	}
	if err := svc.store.DeletePipeByWorkspaceIDServiceID(workspaceID, serviceID); err != nil {
		return err
	}
	return nil
}

func (svc *Service) WorkspaceIntegrations(workspaceID int) ([]pipe.Integration, error) {
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

	var igr []pipe.Integration
	for _, current := range svc.getIntegrations() {
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

func (svc *Service) Run(p *pipe.Pipe) {
	svc.mx.RLock()
	host := svc.pipesApiHost
	svc.mx.RUnlock()

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

	svc.store.LoadLastSync(p)
	p.PipeStatus = pipe.NewPipeStatus(p.WorkspaceID, p.ServiceID, p.ID, host)

	if err = svc.store.SavePipeStatus(p.PipeStatus); err != nil {
		svc.notifyBugsnag(err, p)
		return
	}

	s := pipe.NewExternalService(p.ServiceID, p.WorkspaceID)
	auth, err := svc.store.LoadAuthorization(s.GetWorkspaceID(), s.ID())
	if err != nil {
		svc.notifyBugsnag(err, p)
		return
	}

	if err := s.SetAuthData(auth.Data); err != nil {
		svc.notifyBugsnag(err, p)
		return
	}

	if err = svc.refreshAuthorization(auth); err != nil {
		svc.notifyBugsnag(err, p)
		return
	}

	svc.toggl.WithAuthToken(auth.WorkspaceToken)

	var fErr error
	switch p.ID {
	case integrations.UsersPipe:
		fErr = svc.fetchUsers(p)
	case integrations.ProjectsPipe:
		fErr = svc.fetchProjects(p)
	case integrations.TodoListsPipe:
		fErr = svc.fetchTodoLists(p)
	case integrations.TodosPipe, integrations.TasksPipe:
		fErr = svc.fetchTasks(p)
	case integrations.TimeEntriesPipe:
		fErr = svc.fetchTimeEntries(p)
	default:
		bugsnag.Notify(fmt.Errorf("fetchObjects: Unrecognized pipeID - %s", p.ID))
		return
	}
	if fErr != nil {
		svc.notifyBugsnag(err, p)
		return
	}

	var pErr error
	switch p.ID {
	case integrations.UsersPipe:
		pErr = svc.postUsers(p)
	case integrations.ProjectsPipe:
		pErr = svc.postProjects(p)
	case integrations.TodoListsPipe:
		pErr = svc.postTodoLists(p)
	case integrations.TodosPipe, integrations.TasksPipe:
		pErr = svc.postTasks(p)
	case integrations.TimeEntriesPipe:
		service := pipe.NewExternalService(p.ServiceID, p.WorkspaceID)
		if err := service.SetParams(p.ServiceParams); err != nil {
			log.Printf("could not set service params: %v, reason: %v", p.ID, err)
			break
		}
		auth, err := svc.store.LoadAuthorization(service.GetWorkspaceID(), service.ID())
		if err != nil {
			log.Printf("could not get service auth: %v, reason: %v", p.ID, err)
			break
		}
		if err := service.SetAuthData(auth.Data); err != nil {
			log.Printf("could not set auth data: %v, reason: %v", p.ID, err)
			break
		}
		pErr = svc.postTimeEntries(p, service)
	default:
		bugsnag.Notify(fmt.Errorf("postObjects: Unrecognized pipeID - %s", p.ID))
		return
	}
	if pErr != nil {
		svc.notifyBugsnag(err, p)
		return
	}
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

func (svc *Service) AvailablePipeType(pipeID integrations.PipeID) bool {
	return svc.availablePipeType.MatchString(string(pipeID))
}

func (svc *Service) AvailableServiceType(serviceID integrations.ExternalServiceID) bool {
	return svc.availableServiceType.MatchString(string(serviceID))
}

func (svc *Service) setAuthorizationType(serviceID integrations.ExternalServiceID, authType string) {
	svc.mx.Lock()
	defer svc.mx.Unlock()
	svc.availableAuthTypes[serviceID] = authType
}

func (svc *Service) getAvailableAuthorizations(serviceID integrations.ExternalServiceID) string {
	svc.mx.RLock()
	defer svc.mx.RUnlock()
	return svc.availableAuthTypes[serviceID]
}

func (svc *Service) refreshAuthorization(a *pipe.Authorization) error {
	if svc.getAvailableAuthorizations(a.ServiceID) != pipe.TypeOauth2 {
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
	err := a.SetOauth2Token(token)
	if err != nil {
		return err
	}
	if err := svc.store.SaveAuthorization(a); err != nil {
		return err
	}
	return nil
}

func (svc *Service) notifyBugsnag(err error, p *pipe.Pipe) {
	bugsnag.Notify(err, bugsnag.MetaData{
		"pipe": {
			"ID":            p.ID,
			"Name":          p.Name,
			"ServiceParams": string(p.ServiceParams),
			"WorkspaceID":   p.WorkspaceID,
			"ServiceID":     p.ServiceID,
		},
	})
}

func (svc *Service) loadIntegrations(integrationsConfigPath string) *Service {
	svc.mx.Lock()
	defer svc.mx.Unlock()
	b, err := ioutil.ReadFile(integrationsConfigPath)
	if err != nil {
		log.Fatalf("Could not read integrations.json, reason: %v", err)
	}
	if err := json.Unmarshal(b, &svc.availableIntegrations); err != nil {
		log.Fatalf("Could not parse integrations.json, reason: %v", err)
	}
	return svc
}

func (svc *Service) getIntegrations() []*pipe.Integration {
	svc.mx.RLock()
	defer svc.mx.RUnlock()
	return svc.availableIntegrations
}

func (svc *Service) fillAvailableServices() *Service {
	ids := make([]string, len(svc.availableIntegrations))
	for i := range svc.availableIntegrations {
		ids = append(ids, string(svc.availableIntegrations[i].ID))
	}
	svc.availableServiceType = regexp.MustCompile(strings.Join(ids, "|"))
	return svc
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

	usersResponse, err := svc.getUsers(service)
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
	var idMapping *pipe.IDMapping
	if idMapping, err = svc.store.LoadIDMapping(service.GetWorkspaceID(), service.KeyFor(integrations.UsersPipe)); err != nil {
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

	tasksResponse, err := svc.getTasks(service, integrations.TodoListsPipe) // TODO: WTF?? Why toggl.TodoListsPipe
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

	tasksResponse, err := svc.getTasks(service, integrations.TasksPipe)
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

		if err := svc.store.SaveObject(service, integrations.UsersPipe, response); err != nil {
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

		if err := svc.store.SaveObject(service, integrations.ClientsPipe, response); err != nil {
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

		if err := svc.store.SaveObject(service, integrations.ProjectsPipe, response); err != nil {
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

		if err := svc.store.SaveObject(service, integrations.TodoListsPipe, response); err != nil {
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

		if err := svc.store.SaveObject(service, integrations.TasksPipe, response); err != nil {
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

func (svc *Service) fetchTimeEntries(p *pipe.Pipe) error {
	return nil
}

func (svc *Service) getUsers(s integrations.ExternalService) (*toggl.UsersResponse, error) {
	b, err := svc.store.LoadObject(s, integrations.UsersPipe)
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

func (svc *Service) getClients(s integrations.ExternalService) (*toggl.ClientsResponse, error) {
	b, err := svc.store.LoadObject(s, integrations.ClientsPipe)
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

func (svc *Service) getProjects(s integrations.ExternalService) (*toggl.ProjectsResponse, error) {
	b, err := svc.store.LoadObject(s, integrations.ProjectsPipe)
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

func (svc *Service) getTasks(s integrations.ExternalService, objType integrations.PipeID) (*toggl.TasksResponse, error) {
	b, err := svc.store.LoadObject(s, objType)
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

func (svc *Service) fillAvailablePipeTypes() *Service {
	svc.mx.Lock()
	defer svc.mx.Unlock()
	svc.availablePipeType = regexp.MustCompile("users|projects|todolists|todos|tasks|timeentries")
	return svc
}
