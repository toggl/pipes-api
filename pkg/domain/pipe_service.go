package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/bugsnag/bugsnag-go"
	"github.com/tambet/oauthplain"

	"github.com/toggl/pipes-api/pkg/integration"
	"github.com/toggl/pipes-api/pkg/toggl"
)

type Service struct {
	*AuthorizationFactory

	PipesStorage
	AuthorizationsStorage
	IntegrationsStorage
	IDMappingsStorage
	ImportsStorage

	OAuthProvider
	TogglClient
}

func (svc *Service) Ready() []error {
	errs := make([]error, 0)

	if svc.PipesStorage.IsDown() {
		errs = append(errs, errors.New("database is down"))
	}

	if err := svc.TogglClient.Ping(); err != nil {
		errs = append(errs, err)
	}
	return errs
}

func (svc *Service) GetPipe(workspaceID int, serviceID integration.ID, pipeID integration.PipeID) (*Pipe, error) {
	p := NewPipe(workspaceID, serviceID, pipeID)
	if err := svc.PipesStorage.Load(p); err != nil {
		return nil, err
	}
	var err error
	p.PipeStatus, err = svc.PipesStorage.LoadStatus(workspaceID, serviceID, pipeID)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (svc *Service) CreatePipe(workspaceID int, serviceID integration.ID, pipeID integration.PipeID, params []byte) error {
	p := NewPipe(workspaceID, serviceID, pipeID)

	service := NewExternalService(serviceID, workspaceID)
	err := service.SetParams(params)
	if err != nil {
		return SetParamsError{err}
	}
	p.ServiceParams = params

	if err := svc.PipesStorage.Save(p); err != nil {
		return err
	}
	return nil
}

func (svc *Service) UpdatePipe(workspaceID int, serviceID integration.ID, pipeID integration.PipeID, params []byte) error {
	p := NewPipe(workspaceID, serviceID, pipeID)
	if err := svc.PipesStorage.Load(p); err != nil {
		return err
	}
	if !p.Configured {
		return ErrPipeNotConfigured
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return err
	}
	if err := svc.PipesStorage.Save(p); err != nil {
		return err
	}
	return nil
}

func (svc *Service) DeletePipe(workspaceID int, serviceID integration.ID, pipeID integration.PipeID) error {
	p := NewPipe(workspaceID, serviceID, pipeID)
	if err := svc.PipesStorage.Load(p); err != nil {
		return err
	}
	if err := svc.PipesStorage.Delete(p, workspaceID); err != nil {
		return err
	}
	return nil
}

var ErrNoContent = errors.New("no content")

func (svc *Service) GetServicePipeLog(workspaceID int, serviceID integration.ID, pipeID integration.PipeID) (string, error) {
	pipeStatus, err := svc.PipesStorage.LoadStatus(workspaceID, serviceID, pipeID)
	if err != nil {
		return "", err
	}
	if pipeStatus == nil {
		return "", ErrNoContent
	}
	return pipeStatus.GenerateLog(), nil
}

// TODO: Remove (Probably dead method).
func (svc *Service) ClearIDMappings(workspaceID int, serviceID integration.ID, pipeID integration.PipeID) error {
	p := NewPipe(workspaceID, serviceID, pipeID)
	if err := svc.PipesStorage.Load(p); err != nil {
		return err
	}
	if !p.Configured {
		return ErrPipeNotConfigured
	}
	service := NewExternalService(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}
	auth := svc.AuthorizationFactory.Create(workspaceID, serviceID)
	if err := svc.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
		return err
	}
	if err := auth.Refresh(); err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}
	pipeStatus, err := svc.PipesStorage.LoadStatus(p.WorkspaceID, p.ServiceID, p.ID)
	if err != nil {
		return err
	}

	err = svc.IDMappingsStorage.Delete(p.WorkspaceID, service.KeyFor(p.ID), pipeStatus.Key)
	if err != nil {
		return err
	}
	return nil
}

func (svc *Service) GetServiceUsers(workspaceID int, serviceID integration.ID, forceImport bool) (*toggl.UsersResponse, error) {
	service := NewExternalService(serviceID, workspaceID)
	auth := svc.AuthorizationFactory.Create(workspaceID, serviceID)
	err := svc.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth)
	if err != nil {
		return nil, LoadError{err}
	}
	if err := auth.Refresh(); err != nil {
		return nil, RefreshError{errors.New("oAuth refresh failed!")}
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return nil, err
	}

	usersPipe := NewPipe(workspaceID, serviceID, integration.UsersPipe)
	if err := svc.PipesStorage.Load(usersPipe); err != nil {
		return nil, err
	}
	if usersPipe == nil {
		return nil, ErrPipeNotConfigured
	}
	if err := service.SetParams(usersPipe.ServiceParams); err != nil {
		return nil, SetParamsError{err}
	}

	if forceImport {
		if err := svc.ImportsStorage.DeleteUsersFor(service); err != nil {
			return nil, err
		}
	}

	usersResponse, err := svc.ImportsStorage.LoadUsersFor(service)
	if err != nil {
		return nil, err
	}

	if usersResponse == nil {
		if forceImport {
			go func() {
				fetchErr := svc.FetchUsers(usersPipe)
				if fetchErr != nil {
					log.Print(fetchErr.Error())
				}
			}()
		}
		return nil, ErrNoContent
	}
	return usersResponse, nil
}

func (svc *Service) GetServiceAccounts(workspaceID int, serviceID integration.ID, forceImport bool) (*toggl.AccountsResponse, error) {
	service := NewExternalService(serviceID, workspaceID)
	auth := svc.AuthorizationFactory.Create(workspaceID, serviceID)
	err := svc.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth)
	if err != nil {
		return nil, LoadError{err}
	}
	if err := auth.Refresh(); err != nil {
		return nil, RefreshError{errors.New("oAuth refresh failed!")}
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return nil, err
	}
	if forceImport {
		if err := svc.ImportsStorage.DeleteAccountsFor(service); err != nil {
			return nil, err
		}
	}

	accountsResponse, err := svc.ImportsStorage.LoadAccountsFor(service)
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
			if err := svc.ImportsStorage.SaveAccountsFor(service, response); err != nil {
				log.Print(err.Error())
			}
		}()
		return nil, ErrNoContent
	}
	return accountsResponse, nil
}

func (svc *Service) GetAuthURL(serviceID integration.ID, accountName, callbackURL string) (string, error) {
	config, found := svc.OAuthProvider.OAuth1Configs(serviceID)
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

func (svc *Service) CreateAuthorization(workspaceID int, serviceID integration.ID, workspaceToken string, params AuthParams) error {
	auth := svc.AuthorizationFactory.Create(workspaceID, serviceID)
	auth.WorkspaceToken = workspaceToken

	authType, err := svc.IntegrationsStorage.LoadAuthorizationType(serviceID)
	if err != nil {
		return err
	}
	switch authType {
	case TypeOauth1:
		token, err := svc.OAuthProvider.OAuth1Exchange(serviceID, params.AccountName, params.Token, params.Verifier)
		if err != nil {
			return err
		}
		if err := auth.SetOAuth1Token(token); err != nil {
			return err
		}
	case TypeOauth2:
		if params.Code == "" {
			return errors.New("missing code")
		}
		token, err := svc.OAuthProvider.OAuth2Exchange(serviceID, params.Code)
		if err != nil {
			return err
		}
		if err := auth.SetOAuth2Token(token); err != nil {
			return err
		}
	}

	if err := svc.AuthorizationsStorage.Save(auth); err != nil {
		return err
	}
	return nil
}

func (svc *Service) DeleteAuthorization(workspaceID int, serviceID integration.ID) error {
	if err := svc.AuthorizationsStorage.Delete(workspaceID, serviceID); err != nil {
		return err
	}
	if err := svc.PipesStorage.DeleteByWorkspaceIDServiceID(workspaceID, serviceID); err != nil {
		return err
	}
	return nil
}

func (svc *Service) GetIntegrations(workspaceID int) ([]Integration, error) {
	authorizations, err := svc.AuthorizationsStorage.LoadWorkspaceAuthorizations(workspaceID)
	if err != nil {
		return nil, err
	}
	workspacePipes, err := svc.PipesStorage.LoadAll(workspaceID)
	if err != nil {
		return nil, err
	}
	pipeStatuses, err := svc.PipesStorage.LoadAllStatuses(workspaceID)
	if err != nil {
		return nil, err
	}

	availableIntegrations, err := svc.IntegrationsStorage.LoadIntegrations()
	if err != nil {
		return nil, err
	}

	var resultIntegrations []Integration
	for _, current := range availableIntegrations {

		var ci = current
		ci.AuthURL = svc.OAuthProvider.OAuth2URL(ci.ID)
		ci.Authorized = authorizations[ci.ID]

		var pipes []*Pipe
		for i := range ci.Pipes {

			var p = *ci.Pipes[i]
			key := PipesKey(ci.ID, p.ID)
			var existing = workspacePipes[key]

			if existing != nil {
				p.Automatic = existing.Automatic
				p.Configured = existing.Configured
			}

			p.PipeStatus = pipeStatuses[key]
			pipes = append(pipes, &p)
		}
		ci.Pipes = pipes
		resultIntegrations = append(resultIntegrations, *ci)
	}
	return resultIntegrations, nil
}

func (svc *Service) Synchronize(p *Pipe) {
	var err error
	defer func() {
		if err != nil {
			// If it is JSON marshalling error suppress it for status
			if _, ok := err.(*json.UnmarshalTypeError); ok {
				err = ErrJSONParsing
			}
			p.PipeStatus.AddError(err)
		}
		if e := svc.PipesStorage.SaveStatus(p.PipeStatus); e != nil {
			svc.notifyBugsnag(p, e)
			log.Println(e)
		}
	}()

	svc.PipesStorage.LoadLastSyncFor(p)

	p.PipeStatus = NewPipeStatus(p.WorkspaceID, p.ServiceID, p.ID, p.pipesApiHost)
	if err = svc.PipesStorage.SaveStatus(p.PipeStatus); err != nil {
		svc.notifyBugsnag(p, err)
		return
	}

	auth := svc.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
	if err := svc.AuthorizationsStorage.Load(p.WorkspaceID, p.ServiceID, auth); err != nil {
		svc.notifyBugsnag(p, err)
		return
	}
	if err = auth.Refresh(); err != nil {
		svc.notifyBugsnag(p, err)
		return
	}
	svc.TogglClient.WithAuthToken(auth.WorkspaceToken)

	switch p.ID {
	case integration.UsersPipe:
		svc.syncUsers(p)
	case integration.ProjectsPipe:
		svc.syncProjects(p)
	case integration.TodoListsPipe:
		svc.syncTodoLists(p)
	case integration.TodosPipe, integration.TasksPipe:
		svc.syncTasks(p)
	case integration.TimeEntriesPipe:
		svc.syncTEs(p)
	default:
		bugsnag.Notify(fmt.Errorf("unrecognized pipeID: %s", p.ID))
		return
	}
}

// --------------------------- USERS -------------------------------------------

func (svc *Service) syncUsers(p *Pipe) {
	err := svc.FetchUsers(p)
	if err != nil {
		svc.notifyBugsnag(p, err)
		return
	}

	err = svc.postUsers(p)
	if err != nil {
		svc.notifyBugsnag(p, err)
		return
	}
}

func (svc *Service) FetchUsers(p *Pipe) error {
	service := NewExternalService(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}

	auth := svc.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
	if err := svc.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
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
		auth := svc.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
		if err := svc.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
			log.Printf("could not get service auth: %v, reason: %v", p.ID, err)
			return
		}
		if err := service.SetAuthData(auth.Data); err != nil {
			log.Printf("could not set auth data: %v, reason: %v", p.ID, err)
			return
		}

		if err := svc.ImportsStorage.SaveUsersFor(service, response); err != nil {
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

func (svc *Service) postUsers(p *Pipe) error {
	service := NewExternalService(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}

	auth := svc.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
	if err := svc.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
		return err
	}
	if err := auth.Refresh(); err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}

	usersResponse, err := svc.ImportsStorage.LoadUsersFor(service)
	if err != nil {
		return errors.New("unable to get users from DB")
	}
	if usersResponse == nil {
		return errors.New("service users not found")
	}

	if len(p.UsersSelector.IDs) == 0 {
		return errors.New("unable to get selected users")
	}

	var users []*toggl.User
	for _, userID := range p.UsersSelector.IDs {
		for _, user := range usersResponse.Users {
			if user.ForeignID == strconv.Itoa(userID) {
				user.SendInvitation = p.UsersSelector.SendInvites
				users = append(users, user)
			}
		}
	}

	usersImport, err := svc.TogglClient.PostUsers(integration.UsersPipe, toggl.UsersRequest{Users: users})
	if err != nil {
		return err
	}

	idMapping, err := svc.IDMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(integration.UsersPipe))
	if err != nil {
		return err
	}
	for _, user := range usersImport.WorkspaceUsers {
		idMapping.Data[user.ForeignID] = user.ID
	}
	if err := svc.IDMappingsStorage.Save(idMapping); err != nil {
		return err
	}

	p.PipeStatus.Complete(integration.UsersPipe, usersImport.Notifications, usersImport.Count())
	return nil
}

// --------------------------- PROJECTS ----------------------------------------

func (svc *Service) syncProjects(p *Pipe) {
	err := svc.fetchProjects(p)
	if err != nil {
		svc.notifyBugsnag(p, err)
		return
	}

	err = svc.postProjects(p)
	if err != nil {
		svc.notifyBugsnag(p, err)
		return
	}
}

func (svc *Service) fetchProjects(p *Pipe) error {
	response := toggl.ProjectsResponse{}
	service := NewExternalService(p.ServiceID, p.WorkspaceID)

	defer func() {
		if err := service.SetParams(p.ServiceParams); err != nil {
			log.Printf("could not set service params: %v, reason: %v", p.ID, err)
			return
		}
		auth := svc.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
		if err := svc.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
			log.Printf("could not get service auth: %v, reason: %v", p.ID, err)
			return
		}
		if err := service.SetAuthData(auth.Data); err != nil {
			log.Printf("could not ser service auth data: %v, reason: %v", p.ID, err)
			return
		}

		if err := svc.ImportsStorage.SaveProjectsFor(service, response); err != nil {
			log.Printf("could not save object, workspaceID: %v key: %v, reason: %v", service.GetWorkspaceID(), service.KeyFor(integration.ProjectsPipe), err)
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

	auth := svc.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
	if err := svc.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
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
	if clientsIDMapping, err = svc.IDMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(integration.ClientsPipe)); err != nil {
		response.Error = err.Error()
		return err
	}
	if projectsIDMapping, err = svc.IDMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(integration.ProjectsPipe)); err != nil {
		response.Error = err.Error()
		return err
	}

	for _, project := range response.Projects {
		project.ID = projectsIDMapping.Data[project.ForeignID]
		project.ClientID = clientsIDMapping.Data[project.ForeignClientID]
	}

	return nil
}

func (svc *Service) postProjects(p *Pipe) error {
	service := NewExternalService(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}

	auth := svc.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
	if err := svc.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
		return err
	}
	if err := auth.Refresh(); err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}

	projectsResponse, err := svc.ImportsStorage.LoadProjectsFor(service)
	if err != nil {
		return errors.New("unable to get projects from DB")
	}
	if projectsResponse == nil {
		return errors.New("service projects not found")
	}
	projects := toggl.ProjectRequest{
		Projects: projectsResponse.Projects,
	}
	projectsImport, err := svc.TogglClient.PostProjects(integration.ProjectsPipe, projects)
	if err != nil {
		return err
	}
	var idMapping *IDMapping
	if idMapping, err = svc.IDMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(integration.ProjectsPipe)); err != nil {
		return err
	}
	for _, project := range projectsImport.Projects {
		idMapping.Data[project.ForeignID] = project.ID
	}
	if err := svc.IDMappingsStorage.Save(idMapping); err != nil {
		return err
	}
	p.PipeStatus.Complete(integration.ProjectsPipe, projectsImport.Notifications, projectsImport.Count())
	return nil
}

// --------------------------- TO-DO LISTS -------------------------------------

func (svc *Service) syncTodoLists(p *Pipe) {
	err := svc.fetchTodoLists(p)
	if err != nil {
		svc.notifyBugsnag(p, err)
		return
	}

	err = svc.postTodoLists(p)
	if err != nil {
		svc.notifyBugsnag(p, err)
		return
	}
}

func (svc *Service) fetchTodoLists(p *Pipe) error {
	response := toggl.TasksResponse{}
	service := NewExternalService(p.ServiceID, p.WorkspaceID)

	defer func() {
		if err := service.SetParams(p.ServiceParams); err != nil {
			log.Printf("could not set service params: %v, reason: %v", p.ID, err)
			return
		}
		auth := svc.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
		if err := svc.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
			log.Printf("could not get service auth: %v, reason: %v", p.ID, err)
			return
		}
		if err := service.SetAuthData(auth.Data); err != nil {
			log.Printf("could not set auth data: %v, reason: %v", p.ID, err)
			return
		}

		if err := svc.ImportsStorage.SaveTodoListsFor(service, response); err != nil {
			log.Printf("could not save object, workspaceID: %v key: %v, reason: %v", service.GetWorkspaceID(), service.KeyFor(integration.TodoListsPipe), err)
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
	auth := svc.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
	if err := svc.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
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

	if projectsIDMapping, err = svc.IDMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(integration.ProjectsPipe)); err != nil {
		response.Error = err.Error()
		return err
	}
	if taskIDMapping, err = svc.IDMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(integration.TodoListsPipe)); err != nil {
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

func (svc *Service) postTodoLists(p *Pipe) error {
	service := NewExternalService(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}

	auth := svc.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
	if err := svc.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
		return err
	}
	if err := auth.Refresh(); err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}

	tasksResponse, err := svc.ImportsStorage.LoadTodoListsFor(service)
	if err != nil {
		return errors.New("unable to get tasks from DB")
	}
	if tasksResponse == nil {
		return errors.New("service tasks not found")
	}
	trs, err := svc.TogglClient.AdjustRequestSize(tasksResponse.Tasks, 1)
	if err != nil {
		return err
	}
	var notifications []string
	var count int
	for _, tr := range trs {
		tasksImport, err := svc.TogglClient.PostTodoLists(integration.TasksPipe, tr) // TODO: WTF?? Why toggl.TasksPipe
		if err != nil {
			return err
		}
		idMapping, err := svc.IDMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(integration.TodoListsPipe))
		if err != nil {
			return err
		}
		for _, task := range tasksImport.Tasks {
			idMapping.Data[task.ForeignID] = task.ID
		}
		if err := svc.IDMappingsStorage.Save(idMapping); err != nil {
			return err
		}
		notifications = append(notifications, tasksImport.Notifications...)
		count += tasksImport.Count()
	}
	p.PipeStatus.Complete(integration.TodoListsPipe, notifications, count)
	return nil
}

// --------------------------- TASKS -------------------------------------------

func (svc *Service) syncTasks(p *Pipe) {
	err := svc.fetchTasks(p)
	if err != nil {
		svc.notifyBugsnag(p, err)
		return
	}
	err = svc.postTasks(p)
	if err != nil {
		svc.notifyBugsnag(p, err)
		return
	}
}

func (svc *Service) fetchTasks(p *Pipe) error {
	response := toggl.TasksResponse{}
	service := NewExternalService(p.ServiceID, p.WorkspaceID)
	defer func() {
		if err := service.SetParams(p.ServiceParams); err != nil {
			log.Printf("could not set service params: %v, reason: %v", p.ID, err)
			return
		}

		auth := svc.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
		if err := svc.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
			log.Printf("could not get service auth: %v, reason: %v", p.ID, err)
			return
		}
		if err := service.SetAuthData(auth.Data); err != nil {
			log.Printf("could not set auth data: %v, reason: %v", p.ID, err)
			return
		}

		if err := svc.ImportsStorage.SaveTasksFor(service, response); err != nil {
			log.Printf("could not save object, workspaceID: %v key: %v, reason: %v", service.GetWorkspaceID(), service.KeyFor(integration.TasksPipe), err)
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

	auth := svc.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
	if err := svc.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
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

	if projectsIDMapping, err = svc.IDMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(integration.ProjectsPipe)); err != nil {
		response.Error = err.Error()
		return err
	}
	if taskIDMapping, err = svc.IDMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(integration.TasksPipe)); err != nil {
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

func (svc *Service) postTasks(p *Pipe) error {
	service := NewExternalService(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}
	auth := svc.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
	if err := svc.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
		return err
	}
	if err := auth.Refresh(); err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}

	tasksResponse, err := svc.ImportsStorage.LoadTasksFor(service)
	if err != nil {
		return errors.New("unable to get tasks from DB")
	}
	if tasksResponse == nil {
		return errors.New("service tasks not found")
	}
	trs, err := svc.TogglClient.AdjustRequestSize(tasksResponse.Tasks, 1)
	if err != nil {
		return err
	}
	var notifications []string
	var count int
	for _, tr := range trs {
		tasksImport, err := svc.TogglClient.PostTasks(integration.TasksPipe, tr)
		if err != nil {
			return err
		}
		idMapping, err := svc.IDMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(integration.TasksPipe))
		if err != nil {
			return err
		}
		for _, task := range tasksImport.Tasks {
			idMapping.Data[task.ForeignID] = task.ID
		}
		if err := svc.IDMappingsStorage.Save(idMapping); err != nil {
			return err
		}
		notifications = append(notifications, tasksImport.Notifications...)
		count += tasksImport.Count()
	}
	p.PipeStatus.Complete(p.ID, notifications, count)
	return nil
}

// --------------------------- Time Entries ------------------------------------

func (svc *Service) syncTEs(p *Pipe) {
	service := NewExternalService(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		log.Printf("could not set service params: %v, reason: %v", p.ID, err)
		return
	}

	auth := svc.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
	if err := svc.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
		log.Printf("could not get service auth: %v, reason: %v", p.ID, err)
		return
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		log.Printf("could not set auth data: %v, reason: %v", p.ID, err)
		return
	}

	if err := svc.postTimeEntries(p, service); err != nil {
		svc.notifyBugsnag(p, err)
		return
	}
}

func (svc *Service) postTimeEntries(p *Pipe, service integration.Integration) error {
	usersIDMapping, err := svc.IDMappingsStorage.LoadReversed(service.GetWorkspaceID(), service.KeyFor(integration.UsersPipe))
	if err != nil {
		return err
	}

	tasksIDMapping, err := svc.IDMappingsStorage.LoadReversed(service.GetWorkspaceID(), service.KeyFor(integration.TasksPipe))
	if err != nil {
		return err
	}

	projectsIDMapping, err := svc.IDMappingsStorage.LoadReversed(service.GetWorkspaceID(), service.KeyFor(integration.ProjectsPipe))
	if err != nil {
		return err
	}

	entriesIDMapping, err := svc.IDMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(integration.TimeEntriesPipe))
	if err != nil {
		return err
	}

	if p.LastSync == nil {
		currentTime := time.Now()
		p.LastSync = &currentTime
	}

	timeEntries, err := svc.TogglClient.GetTimeEntries(*p.LastSync, usersIDMapping.GetKeys(), projectsIDMapping.GetKeys())
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

	if err := svc.IDMappingsStorage.Save(entriesIDMapping); err != nil {
		return err
	}
	p.PipeStatus.Complete("timeentries", []string{}, len(timeEntries))
	return nil
}

// -------------------------------- CLIENTS ------------------------------------

func (svc *Service) fetchClients(p *Pipe) error {
	service := NewExternalService(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}
	auth := svc.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
	if err := svc.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
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
		auth := svc.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
		if err := svc.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
			log.Printf("could not get service auth: %v, reason: %v", p.ID, err)
			return
		}
		if err := service.SetAuthData(auth.Data); err != nil {
			log.Printf("could not ser service auth data: %v, reason: %v", p.ID, err)
			return
		}

		if err := svc.ImportsStorage.SaveClientsFor(service, response); err != nil {
			log.Printf("could not save object, workspaceID: %v key: %v, reason: %v", service.GetWorkspaceID(), service.KeyFor(integration.ClientsPipe), err)
			return
		}
	}()
	if err != nil {
		response.Error = err.Error()
		return err
	}
	clientsIDMapping, err := svc.IDMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(integration.ClientsPipe))
	if err != nil {
		response.Error = err.Error()
		return err
	}
	for _, client := range response.Clients {
		client.ID = clientsIDMapping.Data[client.ForeignID]
	}
	return nil
}

func (svc *Service) postClients(p *Pipe) error {
	service := NewExternalService(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}

	auth := svc.AuthorizationFactory.Create(p.WorkspaceID, p.ServiceID)
	if err := svc.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}

	clientsResponse, err := svc.ImportsStorage.LoadClientsFor(service)
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
	clientsImport, err := svc.TogglClient.PostClients(integration.ClientsPipe, clients)
	if err != nil {
		return err
	}
	var idMapping *IDMapping
	if idMapping, err = svc.IDMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(integration.ClientsPipe)); err != nil {
		return err
	}
	for _, client := range clientsImport.Clients {
		idMapping.Data[client.ForeignID] = client.ID
	}
	if err := svc.IDMappingsStorage.Save(idMapping); err != nil {
		return err
	}
	p.PipeStatus.Complete(integration.ClientsPipe, clientsImport.Notifications, clientsImport.Count())
	return nil
}

func (svc *Service) notifyBugsnag(p *Pipe, err error) {
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
