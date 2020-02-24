package pipes

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/bugsnag/bugsnag-go"

	"github.com/toggl/pipes-api/pkg/cfg"
	"github.com/toggl/pipes-api/pkg/integrations"
	"github.com/toggl/pipes-api/pkg/toggl"
)

const maxPayloadSizeBytes = 800 * 1000
const usersPipeID = "users"
const clientsPipeID = "clients"
const projectsPipeID = "projects"
const tasksPipeId = "tasks"
const todoPipeId = "todolists"

func (ps *PipeService) GetAccounts(s integrations.Service) (*toggl.AccountsResponse, error) {
	var result []byte
	rows, err := ps.Storage.Query(`
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

func (ps *PipeService) FetchAccounts(s integrations.Service) error {
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
	_, err = ps.Storage.Exec(`
    INSERT INTO imports(workspace_id, Key, data, created_at)
    VALUES($1, $2, $3, NOW())
  	`, s.GetWorkspaceID(), s.KeyFor("accounts"), b)
	if err != nil {
		bugsnag.Notify(err)
		return err
	}
	return nil
}

func (ps *PipeService) ClearImportFor(s integrations.Service, pipeID string) error {
	_, err := ps.Storage.Exec(`
	    DELETE FROM imports
	    WHERE workspace_id = $1 AND Key = $2
	`, s.GetWorkspaceID(), s.KeyFor(pipeID))
	return err
}

func (ps *PipeService) getObject(s integrations.Service, pipeID string) ([]byte, error) {
	var result []byte
	rows, err := ps.Storage.Query(`
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

func (ps *PipeService) postUsers(p *cfg.Pipe) error {
	s, err := ps.ServiceFor(p)
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

	b, err := ps.TogglService.PostPipesAPI(p.Authorization.WorkspaceToken, usersPipeID, toggl.UsersRequest{Users: users})
	if err != nil {
		return err
	}

	var usersImport toggl.UsersImport
	if err := json.Unmarshal(b, &usersImport); err != nil {
		return err
	}

	var connection *Connection
	if connection, err = ps.ConnectionService.LoadConnection(s, usersPipeID); err != nil {
		return err
	}
	for _, user := range usersImport.WorkspaceUsers {
		connection.Data[user.ForeignID] = user.ID
	}
	if err := ps.ConnectionService.Save(connection); err != nil {
		return err
	}

	p.PipeStatus.Complete(usersPipeID, usersImport.Notifications, usersImport.Count())
	return nil
}

func (ps *PipeService) postClients(p *cfg.Pipe) error {
	service, err := ps.ServiceFor(p)
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
	b, err := ps.TogglService.PostPipesAPI(p.Authorization.WorkspaceToken, clientsPipeID, clients)
	if err != nil {
		return err
	}
	var clientsImport toggl.ClientsImport
	if err := json.Unmarshal(b, &clientsImport); err != nil {
		return err
	}
	var connection *Connection
	if connection, err = ps.ConnectionService.LoadConnection(service, clientsPipeID); err != nil {
		return err
	}
	for _, client := range clientsImport.Clients {
		connection.Data[client.ForeignID] = client.ID
	}
	if err := ps.ConnectionService.Save(connection); err != nil {
		return err
	}
	p.PipeStatus.Complete(clientsPipeID, clientsImport.Notifications, clientsImport.Count())
	return nil
}

func (ps *PipeService) postProjects(p *cfg.Pipe) error {
	s, err := ps.ServiceFor(p)
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

	b, err := ps.TogglService.PostPipesAPI(p.Authorization.WorkspaceToken, projectsPipeID, projects)
	if err != nil {
		return err
	}
	var projectsImport toggl.ProjectsImport
	if err := json.Unmarshal(b, &projectsImport); err != nil {
		return err
	}
	var connection *Connection
	if connection, err = ps.ConnectionService.LoadConnection(s, projectsPipeID); err != nil {
		return err
	}
	for _, project := range projectsImport.Projects {
		connection.Data[project.ForeignID] = project.ID
	}
	if err := ps.ConnectionService.Save(connection); err != nil {
		return err
	}
	p.PipeStatus.Complete(projectsPipeID, projectsImport.Notifications, projectsImport.Count())
	return nil
}

func (ps *PipeService) postTodoLists(p *cfg.Pipe) error {
	s, err := ps.ServiceFor(p)
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
		b, err := ps.TogglService.PostPipesAPI(p.Authorization.WorkspaceToken, tasksPipeId, tr)
		if err != nil {
			return err
		}
		var tasksImport toggl.TasksImport
		if err := json.Unmarshal(b, &tasksImport); err != nil {
			return err
		}
		connection, err := ps.ConnectionService.LoadConnection(s, todoPipeId)
		if err != nil {
			return err
		}
		for _, task := range tasksImport.Tasks {
			connection.Data[task.ForeignID] = task.ID
		}
		if err := ps.ConnectionService.Save(connection); err != nil {
			return err
		}
		notifications = append(notifications, tasksImport.Notifications...)
		count += tasksImport.Count()
	}
	p.PipeStatus.Complete(todoPipeId, notifications, count)
	return nil
}

func (ps *PipeService) postTasks(p *cfg.Pipe) error {
	s, err := ps.ServiceFor(p)
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
		b, err := ps.TogglService.PostPipesAPI(p.Authorization.WorkspaceToken, tasksPipeId, tr)
		if err != nil {
			return err
		}
		var tasksImport toggl.TasksImport
		if err := json.Unmarshal(b, &tasksImport); err != nil {
			return err
		}
		connection, err := ps.ConnectionService.LoadConnection(s, tasksPipeId)
		if err != nil {
			return err
		}

		for _, task := range tasksImport.Tasks {
			connection.Data[task.ForeignID] = task.ID
		}
		if err := ps.ConnectionService.Save(connection); err != nil {
			return err
		}
		notifications = append(notifications, tasksImport.Notifications...)
		count += tasksImport.Count()
	}
	p.PipeStatus.Complete(p.ID, notifications, count)
	return nil
}

func (ps *PipeService) saveObject(p *cfg.Pipe, pipeID string, obj interface{}) error {
	b, err := json.Marshal(obj)
	if err != nil {
		bugsnag.Notify(err)
		return err
	}
	s, err := ps.ServiceFor(p)
	if err != nil {
		return err
	}
	_, err = ps.Storage.Exec(`
	  INSERT INTO imports(workspace_id, Key, data, created_at)
    VALUES($1, $2, $3, NOW())
	`, p.WorkspaceID, s.KeyFor(pipeID), b)
	if err != nil {
		bugsnag.Notify(err)
		return err
	}
	return nil
}

func (ps *PipeService) fetchUsers(p *cfg.Pipe) error {
	s, err := ps.ServiceFor(p)
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

func (ps *PipeService) fetchClients(p *cfg.Pipe) error {
	s, err := ps.ServiceFor(p)
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
	connections, err := ps.ConnectionService.LoadConnection(s, clientsPipeID)
	if err != nil {
		response.Error = err.Error()
		return err
	}
	for _, client := range response.Clients {
		client.ID = connections.Data[client.ForeignID]
	}
	return nil
}

func (ps *PipeService) fetchProjects(p *cfg.Pipe) error {
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

	service, err := ps.ServiceFor(p)
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
	if clientConnections, err = ps.ConnectionService.LoadConnection(service, clientsPipeID); err != nil {
		response.Error = err.Error()
		return err
	}
	if projectConnections, err = ps.ConnectionService.LoadConnection(service, projectsPipeID); err != nil {
		response.Error = err.Error()
		return err
	}

	for _, project := range response.Projects {
		project.ID = projectConnections.Data[project.ForeignID]
		project.ClientID = clientConnections.Data[project.ForeignClientID]
	}

	return nil
}

func (ps *PipeService) fetchTodoLists(p *cfg.Pipe) error {
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

	service, err := ps.ServiceFor(p)
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

	if projectConnections, err = ps.ConnectionService.LoadConnection(service, projectsPipeID); err != nil {
		response.Error = err.Error()
		return err
	}
	if taskConnections, err = ps.ConnectionService.LoadConnection(service, todoPipeId); err != nil {
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

func (ps *PipeService) fetchTasks(p *cfg.Pipe) error {
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

	service, err := ps.ServiceFor(p)
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

	if projectConnections, err = ps.ConnectionService.LoadConnection(service, projectsPipeID); err != nil {
		response.Error = err.Error()
		return err
	}
	if taskConnections, err = ps.ConnectionService.LoadConnection(service, tasksPipeId); err != nil {
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

func (ps *PipeService) GetUsers(s integrations.Service) (*toggl.UsersResponse, error) {
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

func (ps *PipeService) getClients(s integrations.Service) (*toggl.ClientsResponse, error) {
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

func (ps *PipeService) getProjects(s integrations.Service) (*toggl.ProjectsResponse, error) {
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

func (ps *PipeService) getTasks(s integrations.Service, objType string) (*toggl.TasksResponse, error) {
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

func adjustRequestSize(tasks []*toggl.Task, split int) ([]*toggl.TaskRequest, error) {
	var trs []*toggl.TaskRequest
	var size int
	size = len(tasks) / split
	for i := 0; i < split; i++ {
		startIndex := i * size
		endIndex := (i + 1) * size
		if i == split-1 {
			endIndex = len(tasks)
		}
		if endIndex > startIndex {
			t := toggl.TaskRequest{
				Tasks: tasks[startIndex:endIndex],
			}
			trs = append(trs, &t)
		}
	}
	for _, tr := range trs {
		j, err := json.Marshal(tr)
		if err != nil {
			return nil, err
		}
		if len(j) > maxPayloadSizeBytes {
			return adjustRequestSize(tasks, split+1)
		}
	}
	return trs, nil
}

func trimSpacesFromName(ps []*toggl.Project) []*toggl.Project {
	var trimmedPs []*toggl.Project
	for _, p := range ps {
		p.Name = strings.TrimSpace(p.Name)
		if len(p.Name) > 0 {
			trimmedPs = append(trimmedPs, p)
		}
	}
	return trimmedPs
}
