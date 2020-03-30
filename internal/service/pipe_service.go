package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"time"

	goauth2 "code.google.com/p/goauth2/oauth"
	"github.com/bugsnag/bugsnag-go"
	"github.com/tambet/oauthplain"

	"github.com/toggl/pipes-api/pkg/domain"
)

type PipeService struct {
	pipesStorage          domain.PipesStorage
	authorizationsStorage domain.AuthorizationsStorage
	integrationsStorage   domain.IntegrationsStorage
	idMappingsStorage     domain.IDMappingsStorage
	importsStorage        domain.ImportsStorage
	oAuthProvider         domain.OAuthProvider
	togglClient           domain.TogglClient
}

func NewPipeService(pipesStorage domain.PipesStorage, authorizationsStorage domain.AuthorizationsStorage, integrationsStorage domain.IntegrationsStorage, idMappingsStorage domain.IDMappingsStorage, importsStorage domain.ImportsStorage, oAuthProvider domain.OAuthProvider, togglClient domain.TogglClient) *PipeService {
	if pipesStorage == nil {
		panic("PipeService.pipesStorage should not be nil")
	}
	if authorizationsStorage == nil {
		panic("PipeService.authorizationsStorage should not be nil")
	}
	if integrationsStorage == nil {
		panic("PipeService.integrationsStorage should not be nil")
	}
	if idMappingsStorage == nil {
		panic("PipeService.idMappingsStorage should not be nil")
	}
	if importsStorage == nil {
		panic("PipeService.importsStorage should not be nil")
	}
	if oAuthProvider == nil {
		panic("PipeService.oAuthProvider should not be nil")
	}
	if togglClient == nil {
		panic("PipeService.togglClient should not be nil")
	}
	return &PipeService{pipesStorage: pipesStorage, authorizationsStorage: authorizationsStorage, integrationsStorage: integrationsStorage, idMappingsStorage: idMappingsStorage, importsStorage: importsStorage, oAuthProvider: oAuthProvider, togglClient: togglClient}
}

func (svc *PipeService) Ready() []error {
	errs := make([]error, 0)

	if svc.pipesStorage.IsDown() {
		errs = append(errs, errors.New("database is down"))
	}

	if err := svc.togglClient.Ping(); err != nil {
		errs = append(errs, err)
	}
	return errs
}

func (svc *PipeService) GetPipe(workspaceID int, serviceID domain.IntegrationID, pipeID domain.PipeID) (*domain.Pipe, error) {
	p := domain.NewPipe(workspaceID, serviceID, pipeID)
	if err := svc.pipesStorage.Load(p); err != nil {
		return nil, err
	}
	var err error
	p.PipeStatus, err = svc.pipesStorage.LoadStatus(workspaceID, serviceID, pipeID)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (svc *PipeService) CreatePipe(workspaceID int, serviceID domain.IntegrationID, pipeID domain.PipeID, params []byte) error {
	p := domain.NewPipe(workspaceID, serviceID, pipeID)

	service := NewPipeIntegration(serviceID, workspaceID)
	err := service.SetParams(params)
	if err != nil {
		return SetParamsError{err}
	}
	p.ServiceParams = params

	if err := svc.pipesStorage.Save(p); err != nil {
		return err
	}
	return nil
}

func (svc *PipeService) UpdatePipe(workspaceID int, serviceID domain.IntegrationID, pipeID domain.PipeID, params []byte) error {
	p := domain.NewPipe(workspaceID, serviceID, pipeID)
	if err := svc.pipesStorage.Load(p); err != nil {
		return err
	}
	if !p.Configured {
		return ErrPipeNotConfigured
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return err
	}
	if err := svc.pipesStorage.Save(p); err != nil {
		return err
	}
	return nil
}

func (svc *PipeService) DeletePipe(workspaceID int, serviceID domain.IntegrationID, pipeID domain.PipeID) error {
	p := domain.NewPipe(workspaceID, serviceID, pipeID)
	if err := svc.pipesStorage.Load(p); err != nil {
		return err
	}
	if err := svc.pipesStorage.Delete(p, workspaceID); err != nil {
		return err
	}
	return nil
}

var ErrNoContent = errors.New("no content")

func (svc *PipeService) GetServicePipeLog(workspaceID int, serviceID domain.IntegrationID, pipeID domain.PipeID) (string, error) {
	pipeStatus, err := svc.pipesStorage.LoadStatus(workspaceID, serviceID, pipeID)
	if err != nil {
		return "", err
	}
	if pipeStatus == nil {
		return "", ErrNoContent
	}
	return pipeStatus.GenerateLog(), nil
}

// Deprecated: TODO: Remove dead method. It's used only in h4xx0rz(old Backoffice) https://github.com/toggl/support/blob/master/app/controllers/workspaces_controller.rb#L145
func (svc *PipeService) ClearIDMappings(workspaceID int, serviceID domain.IntegrationID, pipeID domain.PipeID) error {
	p := domain.NewPipe(workspaceID, serviceID, pipeID)
	if err := svc.pipesStorage.Load(p); err != nil {
		return err
	}
	if !p.Configured {
		return ErrPipeNotConfigured
	}
	service := NewPipeIntegration(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}
	auth := domain.NewAuthorization(workspaceID, serviceID)
	if err := svc.authorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
		return err
	}
	if err := svc.refresh(auth); err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}
	pipeStatus, err := svc.pipesStorage.LoadStatus(p.WorkspaceID, p.ServiceID, p.ID)
	if err != nil {
		return err
	}

	err = svc.idMappingsStorage.Delete(p.WorkspaceID, service.KeyFor(p.ID), pipeStatus.Key)
	if err != nil {
		return err
	}
	return nil
}

func (svc *PipeService) GetServiceUsers(workspaceID int, serviceID domain.IntegrationID, forceImport bool) (*domain.UsersResponse, error) {
	service := NewPipeIntegration(serviceID, workspaceID)
	auth := domain.NewAuthorization(workspaceID, serviceID)
	err := svc.authorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth)
	if err != nil {
		return nil, LoadError{err}
	}
	if err := svc.refresh(auth); err != nil {
		return nil, RefreshError{errors.New("oAuth refresh failed!")}
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return nil, err
	}

	usersPipe := domain.NewPipe(workspaceID, serviceID, domain.UsersPipe)
	if err := svc.pipesStorage.Load(usersPipe); err != nil {
		return nil, err
	}
	if usersPipe == nil {
		return nil, ErrPipeNotConfigured
	}
	if err := service.SetParams(usersPipe.ServiceParams); err != nil {
		return nil, SetParamsError{err}
	}

	if forceImport {
		if err := svc.importsStorage.DeleteUsersFor(service); err != nil {
			return nil, err
		}
	}

	usersResponse, err := svc.importsStorage.LoadUsersFor(service)
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

func (svc *PipeService) GetServiceAccounts(workspaceID int, serviceID domain.IntegrationID, forceImport bool) (*domain.AccountsResponse, error) {
	service := NewPipeIntegration(serviceID, workspaceID)
	auth := domain.NewAuthorization(workspaceID, serviceID)
	err := svc.authorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth)
	if err != nil {
		return nil, LoadError{err}
	}
	if err := svc.refresh(auth); err != nil {
		return nil, RefreshError{errors.New("oAuth refresh failed!")}
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return nil, err
	}
	if forceImport {
		if err := svc.importsStorage.DeleteAccountsFor(service); err != nil {
			return nil, err
		}
	}

	accountsResponse, err := svc.importsStorage.LoadAccountsFor(service)
	if err != nil {
		return nil, err
	}

	if accountsResponse == nil {
		go func() {
			var response domain.AccountsResponse
			accounts, err := service.Accounts()
			response.Accounts = accounts
			if err != nil {
				response.Error = err.Error()
			}
			if err := svc.importsStorage.SaveAccountsFor(service, response); err != nil {
				log.Print(err.Error())
			}
		}()
		return nil, ErrNoContent
	}
	return accountsResponse, nil
}

func (svc *PipeService) GetAuthURL(serviceID domain.IntegrationID, accountName, callbackURL string) (string, error) {
	config, found := svc.oAuthProvider.OAuth1Configs(serviceID)
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

func (svc *PipeService) CreateAuthorization(workspaceID int, serviceID domain.IntegrationID, workspaceToken string, params domain.AuthParams) error {
	auth := domain.NewAuthorization(workspaceID, serviceID)
	auth.WorkspaceToken = workspaceToken

	authType, err := svc.integrationsStorage.LoadAuthorizationType(serviceID)
	if err != nil {
		return err
	}
	switch authType {
	case domain.TypeOauth1:
		token, err := svc.oAuthProvider.OAuth1Exchange(serviceID, params.AccountName, params.Token, params.Verifier)
		if err != nil {
			return err
		}
		if err := auth.SetOAuth1Token(token); err != nil {
			return err
		}
	case domain.TypeOauth2:
		if params.Code == "" {
			return errors.New("missing code")
		}
		token, err := svc.oAuthProvider.OAuth2Exchange(serviceID, params.Code)
		if err != nil {
			return err
		}
		if err := auth.SetOAuth2Token(token); err != nil {
			return err
		}
	}

	if err := svc.authorizationsStorage.Save(auth); err != nil {
		return err
	}
	return nil
}

func (svc *PipeService) DeleteAuthorization(workspaceID int, serviceID domain.IntegrationID) error {
	if err := svc.authorizationsStorage.Delete(workspaceID, serviceID); err != nil {
		return err
	}
	if err := svc.pipesStorage.DeleteByWorkspaceIDServiceID(workspaceID, serviceID); err != nil {
		return err
	}
	return nil
}

func (svc *PipeService) GetIntegrations(workspaceID int) ([]domain.Integration, error) {
	authorizations, err := svc.authorizationsStorage.LoadWorkspaceAuthorizations(workspaceID)
	if err != nil {
		return nil, err
	}
	workspacePipes, err := svc.pipesStorage.LoadAll(workspaceID)
	if err != nil {
		return nil, err
	}
	pipeStatuses, err := svc.pipesStorage.LoadAllStatuses(workspaceID)
	if err != nil {
		return nil, err
	}

	availableIntegrations, err := svc.integrationsStorage.LoadIntegrations()
	if err != nil {
		return nil, err
	}

	var resultIntegrations []domain.Integration
	for _, current := range availableIntegrations {

		current.AuthURL = svc.oAuthProvider.OAuth2URL(current.ID)
		current.Authorized = authorizations[current.ID]

		var pipes []*domain.Pipe
		for _, pipe := range current.Pipes {
			key := domain.PipesKey(current.ID, pipe.ID)

			var existing = workspacePipes[key]
			if existing != nil {
				pipe.Automatic = existing.Automatic
				pipe.Configured = existing.Configured
			}

			pipe.PipeStatus = pipeStatuses[key]
			pipes = append(pipes, pipe)
		}
		current.Pipes = pipes
		resultIntegrations = append(resultIntegrations, *current)
	}
	return resultIntegrations, nil
}

func (svc *PipeService) Synchronize(p *domain.Pipe) {
	var err error
	defer func() {
		if err != nil {
			// If it is JSON marshalling error suppress it for status
			if _, ok := err.(*json.UnmarshalTypeError); ok {
				err = ErrJSONParsing
			}
			p.PipeStatus.AddError(err)
		}
		if e := svc.pipesStorage.SaveStatus(p.PipeStatus); e != nil {
			svc.notifyBugsnag(p, e)
			log.Println(e)
		}
	}()

	svc.pipesStorage.LoadLastSyncFor(p)

	p.PipeStatus = domain.NewStatus(p.WorkspaceID, p.ServiceID, p.ID, p.PipesApiHost)
	if err = svc.pipesStorage.SaveStatus(p.PipeStatus); err != nil {
		svc.notifyBugsnag(p, err)
		return
	}

	auth := domain.NewAuthorization(p.WorkspaceID, p.ServiceID)
	if err := svc.authorizationsStorage.Load(p.WorkspaceID, p.ServiceID, auth); err != nil {
		svc.notifyBugsnag(p, err)
		return
	}
	if err = svc.refresh(auth); err != nil {
		svc.notifyBugsnag(p, err)
		return
	}
	svc.togglClient.WithAuthToken(auth.WorkspaceToken)

	switch p.ID {
	case domain.UsersPipe:
		svc.syncUsers(p)
	case domain.ProjectsPipe:
		svc.syncProjects(p)
	case domain.TodoListsPipe:
		svc.syncTodoLists(p)
	case domain.TodosPipe, domain.TasksPipe:
		svc.syncTasks(p)
	case domain.TimeEntriesPipe:
		svc.syncTEs(p)
	default:
		bugsnag.Notify(fmt.Errorf("unrecognized pipeID: %s", p.ID))
		return
	}
}

// --------------------------- USERS -------------------------------------------

func (svc *PipeService) syncUsers(p *domain.Pipe) {
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

func (svc *PipeService) FetchUsers(p *domain.Pipe) error {
	service := NewPipeIntegration(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}

	auth := domain.NewAuthorization(p.WorkspaceID, p.ServiceID)
	if err := svc.authorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
		return err
	}

	if err := svc.refresh(auth); err != nil {
		return err
	}

	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}

	users, err := service.Users()
	response := domain.UsersResponse{Users: users}
	defer func() {
		if err := service.SetParams(p.ServiceParams); err != nil {
			log.Printf("could not set service params: %v, reason: %v", p.ID, err)
			return
		}
		auth := domain.NewAuthorization(p.WorkspaceID, p.ServiceID)
		if err := svc.authorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
			log.Printf("could not get service auth: %v, reason: %v", p.ID, err)
			return
		}
		if err := service.SetAuthData(auth.Data); err != nil {
			log.Printf("could not set auth data: %v, reason: %v", p.ID, err)
			return
		}

		if err := svc.importsStorage.SaveUsersFor(service, response); err != nil {
			log.Printf("could not save object, workspaceID: %v key: %v, reason: %v", service.GetWorkspaceID(), service.KeyFor(domain.UsersPipe), err)
			return
		}
	}()
	if err != nil {
		response.Error = err.Error()
		return err
	}
	return nil
}

func (svc *PipeService) postUsers(p *domain.Pipe) error {
	service := NewPipeIntegration(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}

	auth := domain.NewAuthorization(p.WorkspaceID, p.ServiceID)
	if err := svc.authorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
		return err
	}
	if err := svc.refresh(auth); err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}

	usersResponse, err := svc.importsStorage.LoadUsersFor(service)
	if err != nil {
		return errors.New("unable to get users from DB")
	}
	if usersResponse == nil {
		return errors.New("service users not found")
	}

	if len(p.UsersSelector.IDs) == 0 {
		return errors.New("unable to get selected users")
	}

	var users []*domain.User
	for _, userID := range p.UsersSelector.IDs {
		for _, user := range usersResponse.Users {
			if user.ForeignID == strconv.Itoa(userID) {
				user.SendInvitation = p.UsersSelector.SendInvites
				users = append(users, user)
			}
		}
	}

	usersImport, err := svc.togglClient.PostUsers(domain.UsersPipe, domain.UsersRequest{Users: users})
	if err != nil {
		return err
	}

	idMapping, err := svc.idMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(domain.UsersPipe))
	if err != nil {
		return err
	}
	for _, user := range usersImport.WorkspaceUsers {
		idMapping.Data[user.ForeignID] = user.ID
	}
	if err := svc.idMappingsStorage.Save(idMapping); err != nil {
		return err
	}

	p.PipeStatus.Complete(domain.UsersPipe, usersImport.Notifications, usersImport.Count())
	return nil
}

// --------------------------- PROJECTS ----------------------------------------

func (svc *PipeService) syncProjects(p *domain.Pipe) {
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

func (svc *PipeService) fetchProjects(p *domain.Pipe) error {
	response := domain.ProjectsResponse{}
	service := NewPipeIntegration(p.ServiceID, p.WorkspaceID)

	defer func() {
		if err := service.SetParams(p.ServiceParams); err != nil {
			log.Printf("could not set service params: %v, reason: %v", p.ID, err)
			return
		}
		auth := domain.NewAuthorization(p.WorkspaceID, p.ServiceID)
		if err := svc.authorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
			log.Printf("could not get service auth: %v, reason: %v", p.ID, err)
			return
		}
		if err := service.SetAuthData(auth.Data); err != nil {
			log.Printf("could not ser service auth data: %v, reason: %v", p.ID, err)
			return
		}

		if err := svc.importsStorage.SaveProjectsFor(service, response); err != nil {
			log.Printf("could not save object, workspaceID: %v key: %v, reason: %v", service.GetWorkspaceID(), service.KeyFor(domain.ProjectsPipe), err)
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

	auth := domain.NewAuthorization(p.WorkspaceID, p.ServiceID)
	if err := svc.authorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
		return err
	}
	if err := svc.refresh(auth); err != nil {
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

	var clientsIDMapping, projectsIDMapping *domain.IDMapping
	if clientsIDMapping, err = svc.idMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(domain.ClientsPipe)); err != nil {
		response.Error = err.Error()
		return err
	}
	if projectsIDMapping, err = svc.idMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(domain.ProjectsPipe)); err != nil {
		response.Error = err.Error()
		return err
	}

	for _, project := range response.Projects {
		project.ID = projectsIDMapping.Data[project.ForeignID]
		project.ClientID = clientsIDMapping.Data[project.ForeignClientID]
	}

	return nil
}

func (svc *PipeService) postProjects(p *domain.Pipe) error {
	service := NewPipeIntegration(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}

	auth := domain.NewAuthorization(p.WorkspaceID, p.ServiceID)
	if err := svc.authorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
		return err
	}
	if err := svc.refresh(auth); err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}

	projectsResponse, err := svc.importsStorage.LoadProjectsFor(service)
	if err != nil {
		return errors.New("unable to get projects from DB")
	}
	if projectsResponse == nil {
		return errors.New("service projects not found")
	}
	projects := domain.ProjectRequest{
		Projects: projectsResponse.Projects,
	}
	projectsImport, err := svc.togglClient.PostProjects(domain.ProjectsPipe, projects)
	if err != nil {
		return err
	}
	var idMapping *domain.IDMapping
	if idMapping, err = svc.idMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(domain.ProjectsPipe)); err != nil {
		return err
	}
	for _, project := range projectsImport.Projects {
		idMapping.Data[project.ForeignID] = project.ID
	}
	if err := svc.idMappingsStorage.Save(idMapping); err != nil {
		return err
	}
	p.PipeStatus.Complete(domain.ProjectsPipe, projectsImport.Notifications, projectsImport.Count())
	return nil
}

// --------------------------- TO-DO LISTS -------------------------------------

func (svc *PipeService) syncTodoLists(p *domain.Pipe) {
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

func (svc *PipeService) fetchTodoLists(p *domain.Pipe) error {
	response := domain.TasksResponse{}
	service := NewPipeIntegration(p.ServiceID, p.WorkspaceID)

	defer func() {
		if err := service.SetParams(p.ServiceParams); err != nil {
			log.Printf("could not set service params: %v, reason: %v", p.ID, err)
			return
		}
		auth := domain.NewAuthorization(p.WorkspaceID, p.ServiceID)
		if err := svc.authorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
			log.Printf("could not get service auth: %v, reason: %v", p.ID, err)
			return
		}
		if err := service.SetAuthData(auth.Data); err != nil {
			log.Printf("could not set auth data: %v, reason: %v", p.ID, err)
			return
		}

		if err := svc.importsStorage.SaveTodoListsFor(service, response); err != nil {
			log.Printf("could not save object, workspaceID: %v key: %v, reason: %v", service.GetWorkspaceID(), service.KeyFor(domain.TodoListsPipe), err)
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
	auth := domain.NewAuthorization(p.WorkspaceID, p.ServiceID)
	if err := svc.authorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
		return err
	}
	if err := svc.refresh(auth); err != nil {
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

	var projectsIDMapping, taskIDMapping *domain.IDMapping

	if projectsIDMapping, err = svc.idMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(domain.ProjectsPipe)); err != nil {
		response.Error = err.Error()
		return err
	}
	if taskIDMapping, err = svc.idMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(domain.TodoListsPipe)); err != nil {
		response.Error = err.Error()
		return err
	}

	response.Tasks = make([]*domain.Task, 0)
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

func (svc *PipeService) postTodoLists(p *domain.Pipe) error {
	service := NewPipeIntegration(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}

	auth := domain.NewAuthorization(p.WorkspaceID, p.ServiceID)
	if err := svc.authorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
		return err
	}
	if err := svc.refresh(auth); err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}

	tasksResponse, err := svc.importsStorage.LoadTodoListsFor(service)
	if err != nil {
		return errors.New("unable to get tasks from DB")
	}
	if tasksResponse == nil {
		return errors.New("service tasks not found")
	}
	trs, err := svc.togglClient.AdjustRequestSize(tasksResponse.Tasks, 1)
	if err != nil {
		return err
	}
	var notifications []string
	var count int
	for _, tr := range trs {
		tasksImport, err := svc.togglClient.PostTodoLists(domain.TasksPipe, tr) // TODO: WTF?? Why toggl.TasksPipe
		if err != nil {
			return err
		}
		idMapping, err := svc.idMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(domain.TodoListsPipe))
		if err != nil {
			return err
		}
		for _, task := range tasksImport.Tasks {
			idMapping.Data[task.ForeignID] = task.ID
		}
		if err := svc.idMappingsStorage.Save(idMapping); err != nil {
			return err
		}
		notifications = append(notifications, tasksImport.Notifications...)
		count += tasksImport.Count()
	}
	p.PipeStatus.Complete(domain.TodoListsPipe, notifications, count)
	return nil
}

// --------------------------- TASKS -------------------------------------------

func (svc *PipeService) syncTasks(p *domain.Pipe) {
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

func (svc *PipeService) fetchTasks(p *domain.Pipe) error {
	response := domain.TasksResponse{}
	service := NewPipeIntegration(p.ServiceID, p.WorkspaceID)
	defer func() {
		if err := service.SetParams(p.ServiceParams); err != nil {
			log.Printf("could not set service params: %v, reason: %v", p.ID, err)
			return
		}

		auth := domain.NewAuthorization(p.WorkspaceID, p.ServiceID)
		if err := svc.authorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
			log.Printf("could not get service auth: %v, reason: %v", p.ID, err)
			return
		}
		if err := service.SetAuthData(auth.Data); err != nil {
			log.Printf("could not set auth data: %v, reason: %v", p.ID, err)
			return
		}

		if err := svc.importsStorage.SaveTasksFor(service, response); err != nil {
			log.Printf("could not save object, workspaceID: %v key: %v, reason: %v", service.GetWorkspaceID(), service.KeyFor(domain.TasksPipe), err)
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

	auth := domain.NewAuthorization(p.WorkspaceID, p.ServiceID)
	if err := svc.authorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
		return err
	}
	if err := svc.refresh(auth); err != nil {
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
	var projectsIDMapping, taskIDMapping *domain.IDMapping

	if projectsIDMapping, err = svc.idMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(domain.ProjectsPipe)); err != nil {
		response.Error = err.Error()
		return err
	}
	if taskIDMapping, err = svc.idMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(domain.TasksPipe)); err != nil {
		response.Error = err.Error()
		return err
	}

	response.Tasks = make([]*domain.Task, 0)
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

func (svc *PipeService) postTasks(p *domain.Pipe) error {
	service := NewPipeIntegration(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}
	auth := domain.NewAuthorization(p.WorkspaceID, p.ServiceID)
	if err := svc.authorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
		return err
	}
	if err := svc.refresh(auth); err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}

	tasksResponse, err := svc.importsStorage.LoadTasksFor(service)
	if err != nil {
		return errors.New("unable to get tasks from DB")
	}
	if tasksResponse == nil {
		return errors.New("service tasks not found")
	}
	trs, err := svc.togglClient.AdjustRequestSize(tasksResponse.Tasks, 1)
	if err != nil {
		return err
	}
	var notifications []string
	var count int
	for _, tr := range trs {
		tasksImport, err := svc.togglClient.PostTasks(domain.TasksPipe, tr)
		if err != nil {
			return err
		}
		idMapping, err := svc.idMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(domain.TasksPipe))
		if err != nil {
			return err
		}
		for _, task := range tasksImport.Tasks {
			idMapping.Data[task.ForeignID] = task.ID
		}
		if err := svc.idMappingsStorage.Save(idMapping); err != nil {
			return err
		}
		notifications = append(notifications, tasksImport.Notifications...)
		count += tasksImport.Count()
	}
	p.PipeStatus.Complete(p.ID, notifications, count)
	return nil
}

// --------------------------- Time Entries ------------------------------------

func (svc *PipeService) syncTEs(p *domain.Pipe) {
	service := NewPipeIntegration(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		log.Printf("could not set service params: %v, reason: %v", p.ID, err)
		return
	}

	auth := domain.NewAuthorization(p.WorkspaceID, p.ServiceID)
	if err := svc.authorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
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

func (svc *PipeService) postTimeEntries(p *domain.Pipe, service domain.PipeIntegration) error {
	usersIDMapping, err := svc.idMappingsStorage.LoadReversed(service.GetWorkspaceID(), service.KeyFor(domain.UsersPipe))
	if err != nil {
		return err
	}

	tasksIDMapping, err := svc.idMappingsStorage.LoadReversed(service.GetWorkspaceID(), service.KeyFor(domain.TasksPipe))
	if err != nil {
		return err
	}

	projectsIDMapping, err := svc.idMappingsStorage.LoadReversed(service.GetWorkspaceID(), service.KeyFor(domain.ProjectsPipe))
	if err != nil {
		return err
	}

	entriesIDMapping, err := svc.idMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(domain.TimeEntriesPipe))
	if err != nil {
		return err
	}

	if p.LastSync == nil {
		currentTime := time.Now()
		p.LastSync = &currentTime
	}

	timeEntries, err := svc.togglClient.GetTimeEntries(*p.LastSync, usersIDMapping.GetKeys(), projectsIDMapping.GetKeys())
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
					"IntegrationID": service.GetWorkspaceID(),
				},
				"Entry": {
					"IntegrationID": entry.ID,
					"TaskID":        entry.TaskID,
					"UserID":        entry.UserID,
					"ProjectID":     entry.ProjectID,
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

	if err := svc.idMappingsStorage.Save(entriesIDMapping); err != nil {
		return err
	}
	p.PipeStatus.Complete("timeentries", []string{}, len(timeEntries))
	return nil
}

// -------------------------------- CLIENTS ------------------------------------

func (svc *PipeService) fetchClients(p *domain.Pipe) error {
	service := NewPipeIntegration(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}
	auth := domain.NewAuthorization(p.WorkspaceID, p.ServiceID)
	if err := svc.authorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
		return err
	}
	if err := svc.refresh(auth); err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}

	clients, err := service.Clients()
	response := domain.ClientsResponse{Clients: clients}
	defer func() {
		if err := service.SetParams(p.ServiceParams); err != nil {
			log.Printf("could not set service params: %v, reason: %v", p.ID, err)
			return
		}
		auth := domain.NewAuthorization(p.WorkspaceID, p.ServiceID)
		if err := svc.authorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
			log.Printf("could not get service auth: %v, reason: %v", p.ID, err)
			return
		}
		if err := service.SetAuthData(auth.Data); err != nil {
			log.Printf("could not ser service auth data: %v, reason: %v", p.ID, err)
			return
		}

		if err := svc.importsStorage.SaveClientsFor(service, response); err != nil {
			log.Printf("could not save object, workspaceID: %v key: %v, reason: %v", service.GetWorkspaceID(), service.KeyFor(domain.ClientsPipe), err)
			return
		}
	}()
	if err != nil {
		response.Error = err.Error()
		return err
	}
	clientsIDMapping, err := svc.idMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(domain.ClientsPipe))
	if err != nil {
		response.Error = err.Error()
		return err
	}
	for _, client := range response.Clients {
		client.ID = clientsIDMapping.Data[client.ForeignID]
	}
	return nil
}

func (svc *PipeService) postClients(p *domain.Pipe) error {
	service := NewPipeIntegration(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}

	auth := domain.NewAuthorization(p.WorkspaceID, p.ServiceID)
	if err := svc.authorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}

	clientsResponse, err := svc.importsStorage.LoadClientsFor(service)
	if err != nil {
		return errors.New("unable to get clients from DB")
	}
	if clientsResponse == nil {
		return errors.New("service clients not found")
	}
	clients := domain.ClientRequest{
		Clients: clientsResponse.Clients,
	}
	if len(clientsResponse.Clients) == 0 {
		return nil
	}
	clientsImport, err := svc.togglClient.PostClients(domain.ClientsPipe, clients)
	if err != nil {
		return err
	}
	var idMapping *domain.IDMapping
	if idMapping, err = svc.idMappingsStorage.Load(service.GetWorkspaceID(), service.KeyFor(domain.ClientsPipe)); err != nil {
		return err
	}
	for _, client := range clientsImport.Clients {
		idMapping.Data[client.ForeignID] = client.ID
	}
	if err := svc.idMappingsStorage.Save(idMapping); err != nil {
		return err
	}
	p.PipeStatus.Complete(domain.ClientsPipe, clientsImport.Notifications, clientsImport.Count())
	return nil
}

func (svc *PipeService) refresh(a *domain.Authorization) error {
	authType, err := svc.integrationsStorage.LoadAuthorizationType(a.ServiceID)
	if err != nil {
		return err
	}
	if authType != domain.TypeOauth2 {
		return nil
	}
	var token goauth2.Token
	if err := json.Unmarshal(a.Data, &token); err != nil {
		return err
	}
	if !token.Expired() {
		return nil
	}
	config, res := svc.oAuthProvider.OAuth2Configs(a.ServiceID)
	if !res {
		return errors.New("service OAuth config not found")
	}
	if err := svc.oAuthProvider.OAuth2Refresh(config, &token); err != nil {
		return fmt.Errorf("unable to refresh oAuth2 token, reason: %w", err)
	}
	if err := a.SetOAuth2Token(&token); err != nil {
		return err
	}
	if err := svc.authorizationsStorage.Save(a); err != nil {
		return err
	}
	return nil
}

func (svc *PipeService) notifyBugsnag(p *domain.Pipe, err error) {
	meta := bugsnag.MetaData{
		"pipe": {
			"IntegrationID": p.ID,
			"Name":          p.Name,
			"ServiceParams": string(p.ServiceParams),
			"WorkspaceID":   p.WorkspaceID,
			"ServiceID":     p.ServiceID,
		},
	}
	log.Println(err, meta)
	bugsnag.Notify(err, meta)
}
