package toggl

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/toggl/pipes-api/pkg/domain"
)

const maxPayloadSizeBytes = 800 * 1000

var (
	ErrApiNotHealthy = errors.New("toggl api is not healthy, got status code")
)

type ApiClient struct {
	URL string
}

func NewApiClient(URL string) *ApiClient {
	if _, err := url.Parse(URL); err != nil {
		panic("ApiClient.URL should be a valid URL")
	}
	return &ApiClient{URL: URL}
}

func (c *ApiClient) GetWorkspaceIdByToken(token string) (int, error) {

	url := fmt.Sprintf("%s/api/pipes/workspace", c.URL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "toggl-pipes")
	req.SetBasicAuth(token, "api_token")
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

	var response domain.WorkspaceResponse
	if err := json.Unmarshal(b, &response); err != nil {
		return 0, err
	}

	return response.Workspace.ID, nil
}

func (c *ApiClient) PostClients(token string, clientsPipeID domain.PipeID, clients interface{}) (*domain.ClientsImport, error) {
	b, err := c.postPipesAPI(token, clientsPipeID, clients)
	if err != nil {
		return nil, err
	}

	clientsImport := new(domain.ClientsImport)
	if err := json.Unmarshal(b, clientsImport); err != nil {
		return nil, err
	}

	return clientsImport, nil
}

func (c *ApiClient) PostProjects(token string, projectsPipeID domain.PipeID, projects interface{}) (*domain.ProjectsImport, error) {
	b, err := c.postPipesAPI(token, projectsPipeID, projects)
	if err != nil {
		return nil, err
	}

	projectsImport := new(domain.ProjectsImport)
	if err := json.Unmarshal(b, projectsImport); err != nil {
		return nil, err
	}

	return projectsImport, nil
}

func (c *ApiClient) PostTasks(token string, tasksPipeID domain.PipeID, tasks interface{}) (*domain.TasksImport, error) {
	b, err := c.postPipesAPI(token, tasksPipeID, tasks)
	if err != nil {
		return nil, err
	}

	var tasksImport *domain.TasksImport
	if err := json.Unmarshal(b, &tasksImport); err != nil {
		return nil, err
	}

	return tasksImport, nil
}

func (c *ApiClient) PostTodoLists(token string, tasksPipeID domain.PipeID, tasks interface{}) (*domain.TasksImport, error) {
	b, err := c.postPipesAPI(token, tasksPipeID, tasks)
	if err != nil {
		return nil, err
	}

	var tasksImport *domain.TasksImport
	if err := json.Unmarshal(b, &tasksImport); err != nil {
		return nil, err
	}

	return tasksImport, nil
}

func (c *ApiClient) PostUsers(token string, usersPipeID domain.PipeID, users interface{}) (*domain.UsersImport, error) {
	b, err := c.postPipesAPI(token, usersPipeID, users)
	if err != nil {
		return nil, err
	}

	var usersImport *domain.UsersImport
	if err := json.Unmarshal(b, &usersImport); err != nil {
		return nil, err
	}
	return usersImport, nil
}

func (c *ApiClient) GetTimeEntries(token string, lastSync time.Time, userIDs, projectsIDs []int) ([]domain.TimeEntry, error) {
	url := fmt.Sprintf("%s/api/pipes/time_entries?since=%d&user_ids=%s&project_ids=%s",
		c.URL, lastSync.Unix(), stringify(userIDs), stringify(projectsIDs))

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "toggl-pipes")
	req.SetBasicAuth(token, "api_token")
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
	var timeEntries []domain.TimeEntry
	if err := json.Unmarshal(b, &timeEntries); err != nil {
		return nil, err
	}
	return timeEntries, nil
}

func (c *ApiClient) Ping() error {
	var client = &http.Client{Timeout: 3 * time.Second}

	url := fmt.Sprintf("%s/api/v9/status", c.URL)
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("error checking toggl api, reason: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %d", ErrApiNotHealthy, resp.StatusCode)
	}
	return nil
}

func (c *ApiClient) AdjustRequestSize(tasks []*domain.Task, split int) ([]*domain.TaskRequest, error) {
	var trs []*domain.TaskRequest
	var size int
	size = len(tasks) / split
	for i := 0; i < split; i++ {
		startIndex := i * size
		endIndex := (i + 1) * size
		if i == split-1 {
			endIndex = len(tasks)
		}
		if endIndex > startIndex {
			t := domain.TaskRequest{
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
			return c.AdjustRequestSize(tasks, split+1)
		}
	}
	return trs, nil
}

func (c *ApiClient) postPipesAPI(token string, pipeID domain.PipeID, payload interface{}) ([]byte, error) {
	start := time.Now()
	url := fmt.Sprintf("%s/api/pipes/%s", c.URL, pipeID)
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
	req.SetBasicAuth(token, "api_token")
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
