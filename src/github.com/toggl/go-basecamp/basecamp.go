package basecamp

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

const (
	userAgent = "go-basecamp"
	baseURL   = "https://basecamp.com/%d/api/v1/%s"
	authURL   = "https://launchpad.37signals.com/authorization.json"
)

type (
	Client struct {
		AccessToken string
	}

	Account struct {
		Id      int    `json:"id"`
		Name    string `json:"name"`
		Href    string `json:"href"`
		Product string `json:"product"`
	}

	Person struct {
		Id    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email_address"`
		Admin bool   `json:"admin"`
	}

	Project struct {
		Id          int    `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Archived    bool   `json:"archived"`
		Starred     bool   `json:"starred"`
	}

	Todo struct {
		Id      int    `json:"id"`
		Content string `json:"content"`
		DueAt   string `json:"due_at"`
	}

	TodoList struct {
		Id             int    `json:"id"`
		Name           string `json:"name"`
		Description    string `json:"description"`
		Completed      bool   `json:"completed"`
		CompletedCount int    `json:"completed_count"`
		RemainingCount int    `json:"remaining_count"`
		ProjectId      int    `json:"project_id"`

		Bucket struct {
			Id   int    `json:"id"`
			Name string `json:"name"`
			Type string `json:"type"`
		}

		Todos struct {
			Remaining []*Todo `json:"remaining"`
			Completed []*Todo `json:"completed"`
		}
	}
)

func (c *Client) GetAccounts() ([]*Account, error) {
	b, err := c.get(authURL)
	if err != nil {
		return nil, err
	}
	var authorizations map[string]interface{}
	if err := json.Unmarshal(b, &authorizations); err != nil {
		return nil, err
	}
	accounts, ok := authorizations["accounts"].([]interface{})
	if !ok {
		return nil, errors.New("'accounts' not found in response JSON")
	}
	var result []*Account
	for _, data := range accounts {
		values := data.(map[string]interface{})
		account := &Account{
			Id:      int(values["id"].(float64)),
			Name:    values["name"].(string),
			Href:    values["href"].(string),
			Product: values["product"].(string),
		}
		if account.Product != "bcx" {
			continue
		}
		result = append(result, account)
	}
	return result, nil
}

func (c *Client) GetPeople(accountID int) ([]*Person, error) {
	url := fmt.Sprintf(baseURL, accountID, "people.json")
	b, err := c.get(url)
	if err != nil {
		return nil, err
	}
	var result []*Person
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetProjects(accountID int) ([]*Project, error) {
	url := fmt.Sprintf(baseURL, accountID, "projects.json")
	b, err := c.get(url)
	if err != nil {
		return nil, err
	}
	var result []*Project
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) GetTodoLists(accountID int) ([]*TodoList, error) {
	return c.fetchTodoLists(accountID, "todolists.json")
}

func (c *Client) GetCompletedTodoLists(accountID int) ([]*TodoList, error) {
	return c.fetchTodoLists(accountID, "todolists/completed.json")
}

func (c *Client) fetchTodoLists(accountID int, listURL string) ([]*TodoList, error) {
	url := fmt.Sprintf(baseURL, accountID, listURL)
	b, err := c.get(url)
	if err != nil {
		return nil, err
	}
	var result []*TodoList
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, err
	}
	for _, todoList := range result {
		if todoList.Bucket.Type == "Project" {
			todoList.ProjectId = todoList.Bucket.Id
		}
	}
	return result, nil
}

func (c *Client) GetTodoList(accountID, projectID, listID int) (*TodoList, error) {
	url := fmt.Sprintf(baseURL, accountID, fmt.Sprintf("projects/%d/todolists/%d.json", projectID, listID))
	b, err := c.get(url)
	if err != nil {
		return nil, err
	}
	var result *TodoList
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) get(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if 200 != resp.StatusCode {
		return b, fmt.Errorf("%s failed with status code %d", url, resp.StatusCode)
	}
	return b, nil
}
