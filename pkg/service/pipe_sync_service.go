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

	"github.com/toggl/pipes-api/pkg/domain"
	"github.com/toggl/pipes-api/pkg/integration"
)

type PipeSyncService struct {
	pipesStorage          domain.PipesStorage
	authorizationsStorage domain.AuthorizationsStorage
	integrationsStorage   domain.IntegrationsStorage
	idMappingsStorage     domain.IDMappingsStorage
	importsStorage        domain.ImportsStorage
	oAuthProvider         domain.OAuthProvider
	togglClient           domain.TogglClient
}

func NewPipeSyncService(pipesStorage domain.PipesStorage, authorizationsStorage domain.AuthorizationsStorage, integrationsStorage domain.IntegrationsStorage, idMappingsStorage domain.IDMappingsStorage, importsStorage domain.ImportsStorage, oAuthProvider domain.OAuthProvider, togglClient domain.TogglClient) *PipeSyncService {
	if pipesStorage == nil {
		panic("PipeSyncService.pipesStorage should not be nil")
	}
	if authorizationsStorage == nil {
		panic("PipeSyncService.authorizationsStorage should not be nil")
	}
	if integrationsStorage == nil {
		panic("PipeSyncService.integrationsStorage should not be nil")
	}
	if idMappingsStorage == nil {
		panic("PipeSyncService.idMappingsStorage should not be nil")
	}
	if importsStorage == nil {
		panic("PipeSyncService.importsStorage should not be nil")
	}
	if oAuthProvider == nil {
		panic("PipeSyncService.oAuthProvider should not be nil")
	}
	if togglClient == nil {
		panic("PipeSyncService.togglClient should not be nil")
	}
	return &PipeSyncService{pipesStorage: pipesStorage, authorizationsStorage: authorizationsStorage, integrationsStorage: integrationsStorage, idMappingsStorage: idMappingsStorage, importsStorage: importsStorage, oAuthProvider: oAuthProvider, togglClient: togglClient}
}

func (svc *PipeSyncService) GetServicePipeLog(workspaceID int, serviceID domain.IntegrationID, pipeID domain.PipeID) (string, error) {
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
func (svc *PipeSyncService) ClearIDMappings(workspaceID int, serviceID domain.IntegrationID, pipeID domain.PipeID) error {
	p := domain.NewPipe(workspaceID, serviceID, pipeID)
	if err := svc.pipesStorage.Load(p); err != nil {
		return err
	}
	if !p.Configured {
		return ErrPipeNotConfigured
	}

	auth, err := svc.refreshAuthorization(p.WorkspaceID, p.ServiceID)
	if err != nil {
		return err
	}

	service := integration.NewPipeIntegration(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
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

func (svc *PipeSyncService) GetServiceUsers(workspaceID int, serviceID domain.IntegrationID, forceImport bool) (*domain.UsersResponse, error) {
	auth, err := svc.refreshAuthorization(workspaceID, serviceID)
	if err != nil {
		return nil, RefreshError{errors.New("oAuth refresh failed")}
	}

	usersPipe := domain.NewPipe(workspaceID, serviceID, domain.UsersPipe)
	if err := svc.pipesStorage.Load(usersPipe); err != nil {
		return nil, fmt.Errorf("unable to load users pipe, reason: %w", err)
	}
	if usersPipe == nil {
		return nil, ErrPipeNotConfigured
	}

	pipeIntegration := integration.NewPipeIntegration(serviceID, workspaceID)
	if err := pipeIntegration.SetAuthData(auth.Data); err != nil {
		return nil, fmt.Errorf("unable to set auth data, reason: %w", err)
	}
	if err := pipeIntegration.SetParams(usersPipe.ServiceParams); err != nil {
		return nil, SetParamsError{err}
	}

	if forceImport {
		if err := svc.importsStorage.DeleteUsersFor(pipeIntegration); err != nil {
			return nil, fmt.Errorf("unable to force delete users for service, reason: %w", err)
		}
	}

	usersResponse, err := svc.importsStorage.LoadUsersFor(pipeIntegration)
	if err != nil {
		return nil, fmt.Errorf("unable to load users for service, reason: %w", err)
	}

	if usersResponse == nil {
		if forceImport {
			go func() {
				fetchErr := svc.fetchUsers(usersPipe, pipeIntegration)
				if fetchErr != nil {
					log.Print(fetchErr.Error())
				}
			}()
		}
		return nil, ErrNoContent
	}
	return usersResponse, nil
}

func (svc *PipeSyncService) GetServiceAccounts(workspaceID int, serviceID domain.IntegrationID, forceImport bool) (*domain.AccountsResponse, error) {
	service := integration.NewPipeIntegration(serviceID, workspaceID)
	auth := domain.NewAuthorization(workspaceID, serviceID)
	auth, err := svc.refreshAuthorization(workspaceID, serviceID)
	if err != nil {
		return nil, RefreshError{errors.New("oAuth refresh failed")}
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

func (svc *PipeSyncService) GetIntegrations(workspaceID int) ([]domain.Integration, error) {
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

func (svc *PipeSyncService) Synchronize(p *domain.Pipe) error {
	svc.pipesStorage.LoadLastSyncFor(p)
	p.PipeStatus = domain.NewStatus(p.WorkspaceID, p.ServiceID, p.ID, p.PipesApiHost)
	if err := svc.pipesStorage.SaveStatus(p.PipeStatus); err != nil {
		return err
	}

	auth, err := svc.refreshAuthorization(p.WorkspaceID, p.ServiceID)
	if err != nil {
		svc.notifyBugSnag(p, err)
		p.PipeStatus.AddError(err)
	}

	pipeIntegration := integration.NewPipeIntegration(p.ServiceID, p.WorkspaceID)
	if err := pipeIntegration.SetParams(p.ServiceParams); err != nil {
		svc.notifyBugSnag(p, err)
		p.PipeStatus.AddError(err)
	}
	if err := pipeIntegration.SetAuthData(auth.Data); err != nil {
		svc.notifyBugSnag(p, err)
		p.PipeStatus.AddError(err)
	}

	// We start pipes synchronization only if there are no errors on initialization steps above.
	if err == nil {
		svc.togglClient.WithAuthToken(auth.WorkspaceToken)
		switch p.ID {
		case domain.UsersPipe:
			err = svc.syncUsers(p, pipeIntegration)
		case domain.ProjectsPipe:
			err = svc.syncProjects(p, pipeIntegration)
		case domain.TodoListsPipe:
			err = svc.syncTodoLists(p, pipeIntegration)
		case domain.TodosPipe, domain.TasksPipe:
			err = svc.syncTasks(p, pipeIntegration)
		case domain.TimeEntriesPipe:
			err = svc.syncTEs(p, pipeIntegration)
		}
		svc.notifyBugSnag(p, err)
		p.PipeStatus.AddError(err)
	}

	if err = svc.pipesStorage.SaveStatus(p.PipeStatus); err != nil {
		return err
	}
	return nil
}

// --------------------------- USERS -------------------------------------------
func (svc *PipeSyncService) syncUsers(p *domain.Pipe, pipeIntegration domain.PipeIntegration) error {
	err := svc.fetchUsers(p, pipeIntegration)
	if err != nil {
		return err
	}

	err = svc.postUsers(p, pipeIntegration)
	if err != nil {
		return err
	}
	return nil
}

func (svc *PipeSyncService) fetchUsers(p *domain.Pipe, pipeIntegration domain.PipeIntegration) error {
	users, err := pipeIntegration.Users()
	response := domain.UsersResponse{Users: users}
	if err != nil {
		response.Error = err.Error()
		svc.notifyBugSnag(p, err)
	}
	return svc.importsStorage.SaveUsersFor(pipeIntegration, response)
}

func (svc *PipeSyncService) postUsers(p *domain.Pipe, pipeIntegration domain.PipeIntegration) error {
	usersResponse, err := svc.importsStorage.LoadUsersFor(pipeIntegration)
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

	idMapping, err := svc.idMappingsStorage.Load(pipeIntegration.GetWorkspaceID(), pipeIntegration.KeyFor(domain.UsersPipe))
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

func (svc *PipeSyncService) syncProjects(p *domain.Pipe, pipeIntegration domain.PipeIntegration) error {
	err := svc.fetchProjects(p, pipeIntegration)
	if err != nil {
		return err
	}

	err = svc.postProjects(p, pipeIntegration)
	if err != nil {
		return err
	}
	return nil
}

func (svc *PipeSyncService) fetchProjects(p *domain.Pipe, pipeIntegration domain.PipeIntegration) error {
	response := domain.ProjectsResponse{}
	if err := svc.syncClients(p, pipeIntegration); err != nil {
		response.Error = err.Error()
		svc.notifyBugSnag(p, err)
		return svc.importsStorage.SaveProjectsFor(pipeIntegration, response)
	}

	pipeIntegration.SetSince(p.LastSync)
	projects, err := pipeIntegration.Projects()
	if err != nil {
		response.Error = err.Error()
		svc.notifyBugSnag(p, err)
		return svc.importsStorage.SaveProjectsFor(pipeIntegration, response)
	}

	response.Projects = trimSpacesFromName(projects)

	var clientsIDMapping, projectsIDMapping *domain.IDMapping
	if clientsIDMapping, err = svc.idMappingsStorage.Load(pipeIntegration.GetWorkspaceID(), pipeIntegration.KeyFor(domain.ClientsPipe)); err != nil {
		response.Error = err.Error()
		svc.notifyBugSnag(p, err)
		return svc.importsStorage.SaveProjectsFor(pipeIntegration, response)
	}
	if projectsIDMapping, err = svc.idMappingsStorage.Load(pipeIntegration.GetWorkspaceID(), pipeIntegration.KeyFor(domain.ProjectsPipe)); err != nil {
		response.Error = err.Error()
		svc.notifyBugSnag(p, err)
		return svc.importsStorage.SaveProjectsFor(pipeIntegration, response)
	}

	for _, project := range response.Projects {
		project.ID = projectsIDMapping.Data[project.ForeignID]
		project.ClientID = clientsIDMapping.Data[project.ForeignClientID]
	}

	return svc.importsStorage.SaveProjectsFor(pipeIntegration, response)
}

func (svc *PipeSyncService) postProjects(p *domain.Pipe, pipeIntegration domain.PipeIntegration) error {
	projectsResponse, err := svc.importsStorage.LoadProjectsFor(pipeIntegration)
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
	if idMapping, err = svc.idMappingsStorage.Load(pipeIntegration.GetWorkspaceID(), pipeIntegration.KeyFor(domain.ProjectsPipe)); err != nil {
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

func (svc *PipeSyncService) syncTodoLists(p *domain.Pipe, pipeIntegration domain.PipeIntegration) error {
	err := svc.fetchTodoLists(p, pipeIntegration)
	if err != nil {
		return err
	}
	err = svc.postTodoLists(p, pipeIntegration)
	if err != nil {
		return err
	}
	return nil
}

func (svc *PipeSyncService) fetchTodoLists(p *domain.Pipe, pipeIntegration domain.PipeIntegration) error {
	response := domain.TasksResponse{}

	if err := svc.syncProjects(p, pipeIntegration); err != nil {
		response.Error = err.Error()
		svc.notifyBugSnag(p, err)
		return svc.importsStorage.SaveTodoListsFor(pipeIntegration, response)
	}

	pipeIntegration.SetSince(p.LastSync)
	tasks, err := pipeIntegration.TodoLists()
	if err != nil {
		response.Error = err.Error()
		svc.notifyBugSnag(p, err)
		return svc.importsStorage.SaveTodoListsFor(pipeIntegration, response)
	}

	var projectsIDMapping, taskIDMapping *domain.IDMapping

	if projectsIDMapping, err = svc.idMappingsStorage.Load(pipeIntegration.GetWorkspaceID(), pipeIntegration.KeyFor(domain.ProjectsPipe)); err != nil {
		response.Error = err.Error()
		svc.notifyBugSnag(p, err)
		return svc.importsStorage.SaveTodoListsFor(pipeIntegration, response)
	}

	if taskIDMapping, err = svc.idMappingsStorage.Load(pipeIntegration.GetWorkspaceID(), pipeIntegration.KeyFor(domain.TodoListsPipe)); err != nil {
		response.Error = err.Error()
		svc.notifyBugSnag(p, err)
		return svc.importsStorage.SaveTodoListsFor(pipeIntegration, response)
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

	return svc.importsStorage.SaveTodoListsFor(pipeIntegration, response)
}

func (svc *PipeSyncService) postTodoLists(p *domain.Pipe, pipeIntegration domain.PipeIntegration) error {
	tasksResponse, err := svc.importsStorage.LoadTodoListsFor(pipeIntegration)
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
		idMapping, err := svc.idMappingsStorage.Load(pipeIntegration.GetWorkspaceID(), pipeIntegration.KeyFor(domain.TodoListsPipe))
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

func (svc *PipeSyncService) syncTasks(p *domain.Pipe, pipeIntegration domain.PipeIntegration) error {
	err := svc.fetchTasks(p, pipeIntegration)
	if err != nil {
		return err
	}
	err = svc.postTasks(p, pipeIntegration)
	if err != nil {
		return err
	}
	return nil
}

func (svc *PipeSyncService) fetchTasks(p *domain.Pipe, pipeIntegration domain.PipeIntegration) error {
	response := domain.TasksResponse{}

	if err := svc.syncProjects(p, pipeIntegration); err != nil {
		response.Error = err.Error()
		svc.notifyBugSnag(p, err)
		return svc.importsStorage.SaveTasksFor(pipeIntegration, response)
	}

	pipeIntegration.SetSince(p.LastSync)
	tasks, err := pipeIntegration.Tasks()
	if err != nil {
		response.Error = err.Error()
		svc.notifyBugSnag(p, err)
		return svc.importsStorage.SaveTasksFor(pipeIntegration, response)
	}
	var projectsIDMapping, taskIDMapping *domain.IDMapping

	if projectsIDMapping, err = svc.idMappingsStorage.Load(pipeIntegration.GetWorkspaceID(), pipeIntegration.KeyFor(domain.ProjectsPipe)); err != nil {
		response.Error = err.Error()
		svc.notifyBugSnag(p, err)
		return svc.importsStorage.SaveTasksFor(pipeIntegration, response)
	}

	if taskIDMapping, err = svc.idMappingsStorage.Load(pipeIntegration.GetWorkspaceID(), pipeIntegration.KeyFor(domain.TasksPipe)); err != nil {
		response.Error = err.Error()
		svc.notifyBugSnag(p, err)
		return svc.importsStorage.SaveTasksFor(pipeIntegration, response)
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

	return svc.importsStorage.SaveTasksFor(pipeIntegration, response)
}

func (svc *PipeSyncService) postTasks(p *domain.Pipe, pipeIntegration domain.PipeIntegration) error {
	tasksResponse, err := svc.importsStorage.LoadTasksFor(pipeIntegration)
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
		idMapping, err := svc.idMappingsStorage.Load(pipeIntegration.GetWorkspaceID(), pipeIntegration.KeyFor(domain.TasksPipe))
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

func (svc *PipeSyncService) syncTEs(p *domain.Pipe, pipeIntegration domain.PipeIntegration) error {
	usersIDMapping, err := svc.idMappingsStorage.LoadReversed(p.WorkspaceID, pipeIntegration.KeyFor(domain.UsersPipe))
	if err != nil {
		return err
	}

	tasksIDMapping, err := svc.idMappingsStorage.LoadReversed(p.WorkspaceID, pipeIntegration.KeyFor(domain.TasksPipe))
	if err != nil {
		return err
	}

	projectsIDMapping, err := svc.idMappingsStorage.LoadReversed(p.WorkspaceID, pipeIntegration.KeyFor(domain.ProjectsPipe))
	if err != nil {
		return err
	}

	entriesIDMapping, err := svc.idMappingsStorage.Load(p.WorkspaceID, pipeIntegration.KeyFor(domain.TimeEntriesPipe))
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

		entryID, err := pipeIntegration.ExportTimeEntry(&entry)
		if err != nil {
			bugsnag.Notify(err, bugsnag.MetaData{
				"Workspace": {
					"IntegrationID": p.WorkspaceID,
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

	p.PipeStatus.Complete(domain.TimeEntriesPipe, []string{}, len(timeEntries))
	return nil
}

// -------------------------------- CLIENTS ------------------------------------

func (svc *PipeSyncService) syncClients(p *domain.Pipe, pipeIntegration domain.PipeIntegration) error {
	if err := svc.fetchClients(p, pipeIntegration); err != nil {
		return err
	}
	if err := svc.postClients(p, pipeIntegration); err != nil {
		return err
	}
	return nil
}

func (svc *PipeSyncService) fetchClients(p *domain.Pipe, pipeIntegration domain.PipeIntegration) error {
	response := domain.ClientsResponse{}

	clients, err := pipeIntegration.Clients()
	if err != nil {
		response.Error = err.Error()
		svc.notifyBugSnag(p, err)
		return svc.importsStorage.SaveClientsFor(pipeIntegration, response)
	}
	response.Clients = clients

	clientsIDMapping, err := svc.idMappingsStorage.Load(pipeIntegration.GetWorkspaceID(), pipeIntegration.KeyFor(domain.ClientsPipe))
	if err != nil {
		response.Error = err.Error()
		svc.notifyBugSnag(p, err)
		return svc.importsStorage.SaveClientsFor(pipeIntegration, response)
	}

	for _, client := range response.Clients {
		client.ID = clientsIDMapping.Data[client.ForeignID]
	}

	return svc.importsStorage.SaveClientsFor(pipeIntegration, response)
}

func (svc *PipeSyncService) postClients(p *domain.Pipe, pipeIntegration domain.PipeIntegration) error {
	clientsResponse, err := svc.importsStorage.LoadClientsFor(pipeIntegration)
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
	if idMapping, err = svc.idMappingsStorage.Load(pipeIntegration.GetWorkspaceID(), pipeIntegration.KeyFor(domain.ClientsPipe)); err != nil {
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

func (svc *PipeSyncService) refreshAuthorization(workspaceID int, serviceID domain.IntegrationID) (*domain.Authorization, error) {
	auth := domain.NewAuthorization(workspaceID, serviceID)
	if err := svc.authorizationsStorage.Load(workspaceID, serviceID, auth); err != nil {
		return auth, err
	}
	authType, err := svc.integrationsStorage.LoadAuthorizationType(auth.ServiceID)
	if err != nil {
		return auth, err
	}
	if authType != domain.TypeOauth2 {
		return auth, err
	}
	var token goauth2.Token
	if err := json.Unmarshal(auth.Data, &token); err != nil {
		return auth, err
	}
	if !token.Expired() {
		return auth, err
	}
	config, res := svc.oAuthProvider.OAuth2Configs(auth.ServiceID)
	if !res {
		return auth, errors.New("service OAuth config not found")
	}
	if err := svc.oAuthProvider.OAuth2Refresh(config, &token); err != nil {
		return auth, fmt.Errorf("unable to refresh oAuth2 token, reason: %w", err)
	}
	if err := auth.SetOAuth2Token(&token); err != nil {
		return auth, err
	}
	if err := svc.authorizationsStorage.Save(auth); err != nil {
		return auth, err
	}
	return auth, nil
}

func (svc *PipeSyncService) notifyBugSnag(p *domain.Pipe, err error) {
	if err == nil {
		return
	}

	log.Println(err)
	meta := bugsnag.MetaData{
		"pipe": {
			"IntegrationID": p.ID,
			"ServiceParams": string(p.ServiceParams),
			"WorkspaceID":   p.WorkspaceID,
			"ServiceID":     p.ServiceID,
		},
	}
	bugsnag.Notify(err, meta)
}
