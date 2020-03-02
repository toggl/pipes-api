package toggl

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	UsersPipeID       = "users"
	ClientsPipeID     = "clients"
	ProjectsPipeID    = "projects"
	TasksPipeID       = "tasks"
	TodoPipeID        = "todolists"
	TimeEntriesPipeID = "time_entries"
)

var (
	ErrApiNotHealthy = errors.New("toggl api is not healthy, got status code")
)

type ApiClient struct {
	togglApiUrl string

	autoToken string
	mx        sync.Mutex
}

func NewApiClient(url string) *ApiClient {
	return &ApiClient{togglApiUrl: url}
}

func (c *ApiClient) WithAuthToken(authToken string) *ApiClient {
	c.mx.Lock()
	defer c.mx.Unlock()

	c.autoToken = authToken
	return c
}

func (c *ApiClient) GetWorkspaceIdByToken(token string) (int, error) {
	c.WithAuthToken(token)

	url := fmt.Sprintf("%s/api/pipes/workspace", c.togglApiUrl)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "toggl-pipes")
	req.SetBasicAuth(c.autoToken, "api_token")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	var b []byte
	defer resp.Body.Close()
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	if http.StatusOK != resp.StatusCode {
		return 0, fmt.Errorf("GET workspace failed %d", resp.StatusCode)
	}

	var response WorkspaceResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return 0, err
	}

	return response.Workspace.ID, nil
}

func (c *ApiClient) PostClients(clientsPipeID string, clients interface{}) (*ClientsImport, error) {
	b, err := c.postPipesAPI(clientsPipeID, clients)
	if err != nil {
		return nil, err
	}

	clientsImport := new(ClientsImport)
	if err := json.Unmarshal(b, clientsImport); err != nil {
		return nil, err
	}

	return clientsImport, nil
}

func (c *ApiClient) PostProjects(projectsPipeID string, projects interface{}) (*ProjectsImport, error) {
	b, err := c.postPipesAPI(projectsPipeID, projects)
	if err != nil {
		return nil, err
	}

	projectsImport := new(ProjectsImport)
	if err := json.Unmarshal(b, projectsImport); err != nil {
		return nil, err
	}

	return projectsImport, nil
}

func (c *ApiClient) PostTasks(tasksPipeID string, tasks interface{}) (*TasksImport, error) {
	b, err := c.postPipesAPI(tasksPipeID, tasks)
	if err != nil {
		return nil, err
	}

	var tasksImport *TasksImport
	if err := json.Unmarshal(b, &tasksImport); err != nil {
		return nil, err
	}

	return tasksImport, nil
}

func (c *ApiClient) PostTodoLists(tasksPipeID string, tasks interface{}) (*TasksImport, error) {
	b, err := c.postPipesAPI(tasksPipeID, tasks)
	if err != nil {
		return nil, err
	}

	var tasksImport *TasksImport
	if err := json.Unmarshal(b, &tasksImport); err != nil {
		return nil, err
	}

	return tasksImport, nil
}

func (c *ApiClient) PostUsers(usersPipeID string, users interface{}) (*UsersImport, error) {
	b, err := c.postPipesAPI(usersPipeID, users)
	if err != nil {
		return nil, err
	}

	var usersImport *UsersImport
	if err := json.Unmarshal(b, &usersImport); err != nil {
		return nil, err
	}
	return usersImport, nil
}

func (c *ApiClient) GetTimeEntries(lastSync time.Time, userIDs, projectsIDs []int) ([]TimeEntry, error) {
	url := fmt.Sprintf("%s/api/pipes/time_entries?since=%d&user_ids=%s&project_ids=%s",
		c.togglApiUrl, lastSync.Unix(), stringify(userIDs), stringify(projectsIDs))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "toggl-pipes")
	req.SetBasicAuth(c.autoToken, "api_token")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	var b []byte
	defer resp.Body.Close()
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if http.StatusOK != resp.StatusCode {
		return nil, fmt.Errorf("GET time_entries failed %d", resp.StatusCode)
	}
	var timeEntries []TimeEntry
	if err := json.Unmarshal(b, &timeEntries); err != nil {
		return nil, err
	}
	return timeEntries, nil
}

func (c *ApiClient) Ping() error {
	var client = &http.Client{Timeout: 3 * time.Second}

	url := fmt.Sprintf("%s/api/v9/status", c.togglApiUrl)
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("error checking toggl api, reason: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %d", ErrApiNotHealthy, resp.StatusCode)
	}
	return nil
}

func (c *ApiClient) postPipesAPI(pipeID string, payload interface{}) ([]byte, error) {
	start := time.Now()
	url := fmt.Sprintf("%s/api/pipes/%s", c.togglApiUrl, pipeID)
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(b)
	req, err := http.NewRequest("POST", url, buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "toggl-pipes")
	req.SetBasicAuth(c.autoToken, "api_token")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if http.StatusOK != resp.StatusCode {
		return b, fmt.Errorf("%s failed with status code %d", url, resp.StatusCode)
	}
	log.Println("Toggl request", url, "time", time.Since(start))
	return b, nil
}

func stringify(values []int) string {
	s := make([]string, 0, len(values))
	for _, value := range values {
		s = append(s, strconv.Itoa(value))
	}
	return strings.Join(s, ",")
}
