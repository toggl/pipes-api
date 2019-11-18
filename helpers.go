package main

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/bugsnag/bugsnag-go"
)

const maxPayloadSizeBytes = 800 * 1000
const usersPipeID = "users"
const clientsPipeID = "clients"
const projectsPipeID = "projects"
const tasksPipeId = "tasks"
const todoPipeId = "todolists"

func getAccounts(s Service) (*AccountsResponse, error) {
	var result []byte
	rows, err := db.Query(`
		SELECT data FROM imports
		WHERE workspace_id = $1 AND key = $2
		ORDER by created_at DESC
		LIMIT 1
	`, s.WorkspaceID(), s.keyFor("accounts"))
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

	var accountsResponse AccountsResponse
	err = json.Unmarshal(result, &accountsResponse)
	if err != nil {
		return nil, err
	}
	return &accountsResponse, nil
}

func fetchAccounts(s Service) error {
	var response AccountsResponse
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
	_, err = db.Exec(`
    INSERT INTO imports(workspace_id, key, data, created_at)
    VALUES($1, $2, $3, NOW())
  	`, s.WorkspaceID(), s.keyFor("accounts"), b)
	if err != nil {
		bugsnag.Notify(err)
		return err
	}
	return nil
}

func clearImportFor(s Service, pipeID string) error {
	_, err := db.Exec(`
	    DELETE FROM imports
	    WHERE workspace_id = $1 AND key = $2
	`, s.WorkspaceID(), s.keyFor(pipeID))
	return err
}

func getObject(s Service, pipeID string) ([]byte, error) {
	var result []byte
	rows, err := db.Query(`
		SELECT data FROM imports
		WHERE workspace_id = $1 AND key = $2
		ORDER by created_at DESC
		LIMIT 1
	`, s.WorkspaceID(), s.keyFor(pipeID))
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

func getUsers(s Service) (*UsersResponse, error) {
	b, err := getObject(s, usersPipeID)
	if err != nil || b == nil {
		return nil, err
	}

	var usersResponse UsersResponse
	err = json.Unmarshal(b, &usersResponse)
	if err != nil {
		return nil, err
	}
	return &usersResponse, nil
}

func getClients(s Service) (*ClientsResponse, error) {
	b, err := getObject(s, clientsPipeID)
	if err != nil || b == nil {
		return nil, err
	}

	var clientsResponse ClientsResponse
	err = json.Unmarshal(b, &clientsResponse)
	if err != nil {
		return nil, err
	}
	return &clientsResponse, nil
}

func getProjects(s Service) (*ProjectsResponse, error) {
	b, err := getObject(s, projectsPipeID)
	if err != nil || b == nil {
		return nil, err
	}

	var projectsResponse ProjectsResponse
	err = json.Unmarshal(b, &projectsResponse)
	if err != nil {
		return nil, err
	}

	return &projectsResponse, nil
}

func getTasks(s Service, objType string) (*TasksResponse, error) {
	b, err := getObject(s, objType)
	if err != nil || b == nil {
		return nil, err
	}

	var tasksResponse TasksResponse
	err = json.Unmarshal(b, &tasksResponse)
	if err != nil {
		return nil, err
	}
	return &tasksResponse, nil
}

func postUsers(p *Pipe) error {
	s, err := p.Service()
	if err != nil {
		return err
	}
	usersResponse, err := getUsers(s)
	if err != nil {
		return errors.New("unable to get users from DB")
	}
	if usersResponse == nil {
		return errors.New("service users not found")
	}

	var selector Selector
	if err := json.Unmarshal(p.payload, &selector); err != nil {
		return err
	}

	var users []*User
	for _, userID := range selector.IDs {
		for _, user := range usersResponse.Users {
			if user.ForeignID == strconv.Itoa(userID) {
				user.SendInvitation = selector.SendInvites
				users = append(users, user)
			}
		}
	}

	b, err := postPipesAPI(p.authorization.WorkspaceToken, usersPipeID, usersRequest{Users: users})
	if err != nil {
		return err
	}

	var usersImport UsersImport
	if err := json.Unmarshal(b, &usersImport); err != nil {
		return err
	}

	var connection *Connection
	if connection, err = loadConnection(s, usersPipeID); err != nil {
		return err
	}
	for _, user := range usersImport.WorkspaceUsers {
		connection.Data[user.ForeignID] = user.ID
	}
	if err := connection.save(); err != nil {
		return err
	}

	p.PipeStatus.complete(usersPipeID, usersImport.Notifications, usersImport.Count())
	return nil
}

func postClients(p *Pipe) error {
	service, err := p.Service()
	if err != nil {
		return err
	}
	clientsResponse, err := getClients(service)
	if err != nil {
		return errors.New("unable to get clients from DB")
	}
	if clientsResponse == nil {
		return errors.New("service clients not found")
	}
	clients := clientRequest{
		Clients: clientsResponse.Clients,
	}
	if len(clientsResponse.Clients) == 0 {
		return nil
	}
	b, err := postPipesAPI(p.authorization.WorkspaceToken, clientsPipeID, clients)
	if err != nil {
		return err
	}
	var clientsImport ClientsImport
	if err := json.Unmarshal(b, &clientsImport); err != nil {
		return err
	}
	var connection *Connection
	if connection, err = loadConnection(service, clientsPipeID); err != nil {
		return err
	}
	for _, client := range clientsImport.Clients {
		connection.Data[client.ForeignID] = client.ID
	}
	if err := connection.save(); err != nil {
		return err
	}
	p.PipeStatus.complete(clientsPipeID, clientsImport.Notifications, clientsImport.Count())
	return nil
}

func postProjects(p *Pipe) error {
	s, err := p.Service()
	if err != nil {
		return err
	}
	projectsResponse, err := getProjects(s)
	if err != nil {
		return errors.New("unable to get projects from DB")
	}
	if projectsResponse == nil {
		return errors.New("service projects not found")
	}
	projects := projectRequest{
		Projects: projectsResponse.Projects,
	}

	b, err := postPipesAPI(p.authorization.WorkspaceToken, projectsPipeID, projects)
	if err != nil {
		return err
	}
	var projectsImport ProjectsImport
	if err := json.Unmarshal(b, &projectsImport); err != nil {
		return err
	}
	var connection *Connection
	if connection, err = loadConnection(s, projectsPipeID); err != nil {
		return err
	}
	for _, project := range projectsImport.Projects {
		connection.Data[project.ForeignID] = project.ID
	}
	if err := connection.save(); err != nil {
		return err
	}
	p.PipeStatus.complete(projectsPipeID, projectsImport.Notifications, projectsImport.Count())
	return nil
}

func postTodoLists(p *Pipe) error {
	s, err := p.Service()
	if err != nil {
		return err
	}
	tasksResponse, err := getTasks(s, todoPipeId)
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
		b, err := postPipesAPI(p.authorization.WorkspaceToken, tasksPipeId, tr)
		if err != nil {
			return err
		}
		var tasksImport TasksImport
		if err := json.Unmarshal(b, &tasksImport); err != nil {
			return err
		}
		connection, err := loadConnection(s, todoPipeId)
		if err != nil {
			return err
		}
		for _, task := range tasksImport.Tasks {
			connection.Data[task.ForeignID] = task.ID
		}
		if err := connection.save(); err != nil {
			return err
		}
		notifications = append(notifications, tasksImport.Notifications...)
		count += tasksImport.Count()
	}
	p.PipeStatus.complete(todoPipeId, notifications, count)
	return nil
}

func postTasks(p *Pipe) error {
	s, err := p.Service()
	if err != nil {
		return err
	}
	tasksResponse, err := getTasks(s, tasksPipeId)
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
		b, err := postPipesAPI(p.authorization.WorkspaceToken, tasksPipeId, tr)
		if err != nil {
			return err
		}
		var tasksImport TasksImport
		if err := json.Unmarshal(b, &tasksImport); err != nil {
			return err
		}
		connection, err := loadConnection(s, tasksPipeId)
		if err != nil {
			return err
		}

		for _, task := range tasksImport.Tasks {
			connection.Data[task.ForeignID] = task.ID
		}
		if err := connection.save(); err != nil {
			return err
		}
		notifications = append(notifications, tasksImport.Notifications...)
		count += tasksImport.Count()
	}
	p.PipeStatus.complete(p.ID, notifications, count)
	return nil
}

func saveObject(p *Pipe, pipeID string, obj interface{}) error {
	b, err := json.Marshal(obj)
	if err != nil {
		bugsnag.Notify(err)
		return err
	}
	s, err := p.Service()
	if err != nil {
		return err
	}
	_, err = db.Exec(`
	  INSERT INTO imports(workspace_id, key, data, created_at)
    VALUES($1, $2, $3, NOW())
	`, p.workspaceID, s.keyFor(pipeID), b)
	if err != nil {
		bugsnag.Notify(err)
		return err
	}
	return nil
}

func fetchUsers(p *Pipe) error {
	s, err := p.Service()
	if err != nil {
		return err
	}
	users, err := s.Users()
	response := UsersResponse{Users: users}
	defer func() { saveObject(p, usersPipeID, response) }()
	if err != nil {
		response.Error = err.Error()
		return err
	}
	return nil
}

func fetchClients(p *Pipe) error {
	s, err := p.Service()
	if err != nil {
		return err
	}
	clients, err := s.Clients()
	response := ClientsResponse{Clients: clients}
	defer func() { saveObject(p, clientsPipeID, response) }()
	if err != nil {
		response.Error = err.Error()
		return err
	}
	connections, err := loadConnection(s, clientsPipeID)
	if err != nil {
		response.Error = err.Error()
		return err
	}
	for _, client := range response.Clients {
		client.ID = connections.Data[client.ForeignID]
	}
	return nil
}

func fetchProjects(p *Pipe) error {
	response := ProjectsResponse{}
	defer func() { saveObject(p, projectsPipeID, response) }()

	if err := fetchClients(p); err != nil {
		response.Error = err.Error()
		return err
	}
	if err := postClients(p); err != nil {
		response.Error = err.Error()
		return err
	}

	service, err := p.Service()
	if err != nil {
		return err
	}
	service.setSince(p.lastSync)
	projects, err := service.Projects()
	if err != nil {
		response.Error = err.Error()
		return err
	}

	response.Projects = trimSpacesFromName(projects)

	var clientConnections, projectConnections *Connection
	if clientConnections, err = loadConnection(service, clientsPipeID); err != nil {
		response.Error = err.Error()
		return err
	}
	if projectConnections, err = loadConnection(service, projectsPipeID); err != nil {
		response.Error = err.Error()
		return err
	}

	for _, project := range response.Projects {
		project.ID = projectConnections.Data[project.ForeignID]
		project.ClientID = clientConnections.Data[project.foreignClientID]
	}

	return nil
}

func fetchTodoLists(p *Pipe) error {
	response := TasksResponse{}
	defer func() { saveObject(p, todoPipeId, response) }()

	if err := fetchProjects(p); err != nil {
		response.Error = err.Error()
		return err
	}
	if err := postProjects(p); err != nil {
		response.Error = err.Error()
		return err
	}

	service, err := p.Service()
	if err != nil {
		return err
	}
	service.setSince(p.lastSync)
	tasks, err := service.TodoLists()
	if err != nil {
		response.Error = err.Error()
		return err
	}

	var projectConnections, taskConnections *Connection

	if projectConnections, err = loadConnection(service, projectsPipeID); err != nil {
		response.Error = err.Error()
		return err
	}
	if taskConnections, err = loadConnection(service, todoPipeId); err != nil {
		response.Error = err.Error()
		return err
	}

	response.Tasks = make([]*Task, 0)
	for _, task := range tasks {
		id := taskConnections.Data[task.ForeignID]
		if (id > 0) || task.Active {
			task.ID = id
			task.ProjectID = projectConnections.Data[task.foreignProjectID]
			response.Tasks = append(response.Tasks, task)
		}
	}
	return nil
}

func fetchTasks(p *Pipe) error {
	response := TasksResponse{}
	defer func() { saveObject(p, tasksPipeId, response) }()

	if err := fetchProjects(p); err != nil {
		response.Error = err.Error()
		return err
	}
	if err := postProjects(p); err != nil {
		response.Error = err.Error()
		return err
	}

	service, err := p.Service()
	if err != nil {
		return err
	}
	service.setSince(p.lastSync)
	tasks, err := service.Tasks()
	if err != nil {
		response.Error = err.Error()
		return err
	}
	var projectConnections, taskConnections *Connection

	if projectConnections, err = loadConnection(service, projectsPipeID); err != nil {
		response.Error = err.Error()
		return err
	}
	if taskConnections, err = loadConnection(service, tasksPipeId); err != nil {
		response.Error = err.Error()
		return err
	}

	response.Tasks = make([]*Task, 0)
	for _, task := range tasks {
		id := taskConnections.Data[task.ForeignID]
		if (id > 0) || task.Active {
			task.ID = id
			task.ProjectID = projectConnections.Data[task.foreignProjectID]
			response.Tasks = append(response.Tasks, task)
		}
	}
	return nil
}

func adjustRequestSize(tasks []*Task, split int) ([]*taskRequest, error) {
	var trs []*taskRequest
	var size int
	size = len(tasks) / split
	for i := 0; i < split; i++ {
		startIndex := i * size
		endIndex := (i + 1) * size
		if i == split-1 {
			endIndex = len(tasks)
		}
		if endIndex > startIndex {
			t := taskRequest{
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

func trimSpacesFromName(ps []*Project) []*Project {
	var trimmedPs []*Project
	for _, p := range ps {
		p.Name = strings.TrimSpace(p.Name)
		if len(p.Name) > 0 {
			trimmedPs = append(trimmedPs, p)
		}
	}
	return trimmedPs
}

// BugsnagNotifyPipe notifies bugsnag with metadata for the given pipe
func BugsnagNotifyPipe(pipe *Pipe, err error) {
	bugsnag.Notify(err, bugsnag.MetaData{
		"pipe": {
			"ID":            pipe.ID,
			"Name":          pipe.Name,
			"ServiceParams": string(pipe.ServiceParams),
			"workspaceID":   pipe.workspaceID,
			"serviceID":     pipe.serviceID,
		},
	})
	return
}

func numberStrToInt(s string) int {
	if s == "" {
		return 0
	}
	res, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return res
}
