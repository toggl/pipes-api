package main

import (
	"encoding/json"
	"errors"
	"github.com/toggl/bugsnag"
	"strconv"
)

func getAccounts(s Service) (*AccountsResponse, error) {
	var result []byte
	rows, err := db.Query(`
		SELECT data FROM imports
		WHERE workspace_id = $1 AND key = $2
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
	b, err := getObject(s, "users")
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
	b, err := getObject(s, "clients")
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
	b, err := getObject(s, "projects")
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
	s := p.Service()
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
			if user.ForeignID == userID {
				users = append(users, user)
			}
		}
	}

	b, err := postPipesAPI(p.authorization.WorkspaceToken, "users", usersRequest{Users: users})
	if err != nil {
		return err
	}

	var usersImport UsersImport
	if err := json.Unmarshal(b, &usersImport); err != nil {
		return err
	}

	connection := NewConnection(s, "users")
	for _, user := range usersImport.WorkspaceUsers {
		connection.Data[strconv.Itoa(user.ForeignID)] = user.ID
	}
	if err := connection.save(); err != nil {
		return err
	}

	p.PipeStatus.complete("users", usersImport.Notifications, usersImport.Count())
	return nil
}

func postClients(p *Pipe) error {
	service := p.Service()
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
	b, err := postPipesAPI(p.authorization.WorkspaceToken, "clients", clients)
	if err != nil {
		return err
	}
	var clientsImport ClientsImport
	if err := json.Unmarshal(b, &clientsImport); err != nil {
		return err
	}

	connection := NewConnection(service, "clients")
	for _, client := range clientsImport.Clients {
		connection.Data[strconv.Itoa(client.ForeignID)] = client.ID
	}
	if err := connection.save(); err != nil {
		return err
	}
	p.PipeStatus.complete("clients", clientsImport.Notifications, clientsImport.Count())
	return nil
}

func postProjects(p *Pipe) error {
	s := p.Service()
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

	b, err := postPipesAPI(p.authorization.WorkspaceToken, "projects", projects)
	if err != nil {
		return err
	}
	var projectsImport ProjectsImport
	if err := json.Unmarshal(b, &projectsImport); err != nil {
		return err
	}

	connection := NewConnection(s, "projects")
	for _, project := range projectsImport.Projects {
		connection.Data[strconv.Itoa(project.ForeignID)] = project.ID
	}
	if err := connection.save(); err != nil {
		return err
	}
	p.PipeStatus.complete("projects", projectsImport.Notifications, projectsImport.Count())
	return nil
}

func postTodoLists(p *Pipe) error {
	s := p.Service()
	tasksResponse, err := getTasks(s, "todolists")
	if err != nil {
		return errors.New("unable to get tasks from DB")
	}
	if tasksResponse == nil {
		return errors.New("service tasks not found")
	}
	tasks := taskRequest{
		Tasks: tasksResponse.Tasks,
	}
	b, err := postPipesAPI(p.authorization.WorkspaceToken, "tasks", tasks)
	if err != nil {
		return err
	}
	var tasksImport TasksImport
	if err := json.Unmarshal(b, &tasksImport); err != nil {
		return err
	}

	connection := NewConnection(s, "todolists")
	for _, task := range tasksImport.Tasks {
		connection.Data[strconv.Itoa(task.ForeignID)] = task.ID
	}
	if err := connection.save(); err != nil {
		return err
	}
	p.PipeStatus.complete("todolists", tasksImport.Notifications, tasksImport.Count())
	return nil
}

func postTasks(p *Pipe) error {
	s := p.Service()
	tasksResponse, err := getTasks(s, "tasks")
	if err != nil {
		return errors.New("unable to get tasks from DB")
	}
	if tasksResponse == nil {
		return errors.New("service tasks not found")
	}
	tasks := taskRequest{
		Tasks: tasksResponse.Tasks,
	}
	b, err := postPipesAPI(p.authorization.WorkspaceToken, "tasks", tasks)
	if err != nil {
		return err
	}
	var tasksImport TasksImport
	if err := json.Unmarshal(b, &tasksImport); err != nil {
		return err
	}

	connection := NewConnection(s, "tasks")
	for _, task := range tasksImport.Tasks {
		connection.Data[strconv.Itoa(task.ForeignID)] = task.ID
	}
	if err := connection.save(); err != nil {
		return err
	}
	p.PipeStatus.complete("todos", tasksImport.Notifications, tasksImport.Count())
	return nil
}

func saveObject(p *Pipe, pipeID string, obj interface{}) error {
	b, err := json.Marshal(obj)
	if err != nil {
		bugsnag.Notify(err)
		return err
	}
	_, err = db.Exec(`
	  INSERT INTO imports(workspace_id, key, data, created_at)
    VALUES($1, $2, $3, NOW())
	`, p.workspaceID, p.Service().keyFor(pipeID), b)
	if err != nil {
		bugsnag.Notify(err)
		return err
	}
	return nil
}

func fetchUsers(p *Pipe) error {
	users, err := p.Service().Users()
	response := UsersResponse{Users: users}
	defer func() { saveObject(p, "users", response) }()
	if err != nil {
		response.Error = err.Error()
		return err
	}
	return nil
}

func fetchClients(p *Pipe) error {
	clients, err := p.Service().Clients()
	response := ClientsResponse{Clients: clients}
	defer func() { saveObject(p, "clients", response) }()
	if err != nil {
		response.Error = err.Error()
		return err
	}
	connections, err := loadConnection(p.Service(), "clients")
	if err != nil {
		response.Error = err.Error()
		return err
	}
	for _, client := range response.Clients {
		client.ID = connections.Data[strconv.Itoa(client.ForeignID)]
	}
	return nil
}

func fetchProjects(p *Pipe) error {
	response := ProjectsResponse{}
	defer func() { saveObject(p, "projects", response) }()

	if err := fetchClients(p); err != nil {
		response.Error = err.Error()
		return err
	}
	if err := postClients(p); err != nil {
		response.Error = err.Error()
		return err
	}
	projects, err := p.Service().Projects()
	if err != nil {
		response.Error = err.Error()
		return err
	}
	response.Projects = projects

	var clientConnections, projectConnections *Connection
	if clientConnections, err = loadConnection(p.Service(), "clients"); err != nil {
		response.Error = err.Error()
		return err
	}
	if projectConnections, err = loadConnection(p.Service(), "projects"); err != nil {
		response.Error = err.Error()
		return err
	}

	for _, project := range response.Projects {
		project.ID = projectConnections.Data[strconv.Itoa(project.ForeignID)]
		project.ClientID = clientConnections.Data[strconv.Itoa(project.foreignClientID)]
	}

	return nil
}

func fetchTodoLists(p *Pipe) error {
	response := TasksResponse{}
	defer func() { saveObject(p, "todolists", response) }()

	if err := fetchProjects(p); err != nil {
		response.Error = err.Error()
		return err
	}
	if err := postProjects(p); err != nil {
		response.Error = err.Error()
		return err
	}
	tasks, err := p.Service().TodoLists()
	if err != nil {
		response.Error = err.Error()
		return err
	}

	var projectConnections, taskConnections *Connection

	if projectConnections, err = loadConnection(p.Service(), "projects"); err != nil {
		response.Error = err.Error()
		return err
	}
	if taskConnections, err = loadConnection(p.Service(), "todolists"); err != nil {
		response.Error = err.Error()
		return err
	}

	response.Tasks = make([]*Task, 0)
	for _, task := range tasks {
		id := taskConnections.Data[strconv.Itoa(task.ForeignID)]
		if (id > 0) || task.Active {
			task.ID = id
			task.ProjectID = projectConnections.Data[strconv.Itoa(task.foreignProjectID)]
			response.Tasks = append(response.Tasks, task)
		}
	}
	return nil
}

func fetchTasks(p *Pipe) error {
	response := TasksResponse{}
	defer func() { saveObject(p, "tasks", response) }()

	if err := fetchProjects(p); err != nil {
		response.Error = err.Error()
		return err
	}
	if err := postProjects(p); err != nil {
		response.Error = err.Error()
		return err
	}

	tasks, err := p.Service().Tasks()
	if err != nil {
		response.Error = err.Error()
		return err
	}
	var projectConnections, taskConnections *Connection

	if projectConnections, err = loadConnection(p.Service(), "projects"); err != nil {
		response.Error = err.Error()
		return err
	}
	if taskConnections, err = loadConnection(p.Service(), "tasks"); err != nil {
		response.Error = err.Error()
		return err
	}

	response.Tasks = make([]*Task, 0)
	for _, task := range tasks {
		id := taskConnections.Data[strconv.Itoa(task.ForeignID)]
		if (id > 0) || task.Active {
			task.ID = id
			task.ProjectID = projectConnections.Data[strconv.Itoa(task.foreignProjectID)]
			response.Tasks = append(response.Tasks, task)
		}
	}
	return nil
}
