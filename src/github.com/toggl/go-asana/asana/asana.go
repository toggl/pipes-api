package asana

import (
	"encoding/json"
	"fmt"
	"github.com/google/go-querystring/query"
	"net/http"
	"net/url"
	"reflect"
)

const (
	libraryVersion = "0.1"
	userAgent      = "go-asana/" + libraryVersion
	defaultBaseURL = "https://app.asana.com/api/1.0/"
)

var defaultOptFields = map[string]string{
	"tags":       "name,color,notes",
	"users":      "name,email,photo",
	"projects":   "name,color,archived",
	"workspaces": "name,is_organization",
	"tasks":      "name,assignee,assignee_status,completed,parent",
}

type (
	Client struct {
		client    *http.Client
		BaseURL   *url.URL
		UserAgent string
	}

	Workspace struct {
		ID           int64  `json:"id,omitempty"`
		Name         string `json:"name,omitempty"`
		Organization bool   `json:"is_organization,omitempty"`
	}

	User struct {
		ID         int64       `json:"id,omitempty"`
		Email      string      `json:"email,omitempty"`
		Name       string      `json:"name,omitempty"`
		Photo      string      `json:"photo,omitempty"`
		Workspaces []Workspace `json:"workspaces,omitempty"`
	}

	Project struct {
		ID       int64  `json:"id,omitempty"`
		Name     string `json:"name,omitempty"`
		Archived bool   `json:"archived,omitempty"`
		Color    string `json:"color,omitempty"`
		Notes    string `json:"notes,omitempty"`
	}

	Task struct {
		ID             int64     `json:"id,omitempty"`
		Assignee       *User     `json:"assignee,omitempty"`
		AssigneeStatus string    `json:"assignee_status,omitempty"`
		Completed      bool      `json:"completed,omitempty"`
		Name           string    `json:"name,omitempty"`
		Notes          string    `json:"notes,omitempty"`
		ParentTask     *Task     `json:"parent,omitempty"`
		Projects       []Project `json:"projects,omitempty"`
	}

	Tag struct {
		ID    int64  `json:"id,omitempty"`
		Name  string `json:"name,omitempty"`
		Color string `json:"color,omitempty"`
		Notes string `json:"notes,omitempty"`
	}

	Filter struct {
		Archived       bool   `url:"archived,omitempty"`
		Assignee       int64  `url:"assignee,omitempty"`
		Project        int64  `url:"project,omitempty"`
		Workspace      int64  `url:"workspace,omitempty"`
		CompletedSince string `url:"completed_since,omitempty"`
		ModifiedSince  string `url:"modified_since,omitempty"`
		OptFields      string `url:"opt_fields,omitempty"`
	}

	Response struct {
		Data   interface{} `json:"data,omitempty"`
		Errors []Error     `json:"errors,omitempty"`
	}

	Error struct {
		Phrase  string `json:"phrase,omitempty"`
		Message string `json:"message,omitempty"`
	}
)

func (e Error) Error() string {
	return fmt.Sprintf("%v - %v", e.Message, e.Phrase)
}

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	baseURL, _ := url.Parse(defaultBaseURL)
	client := &Client{client: httpClient, BaseURL: baseURL, UserAgent: userAgent}
	return client
}

func (c *Client) ListWorkspaces() ([]Workspace, error) {
	workspaces := new([]Workspace)
	err := c.Request("workspaces", nil, workspaces)
	return *workspaces, err
}

func (c *Client) ListUsers(opt *Filter) ([]User, error) {
	users := new([]User)
	err := c.Request("users", opt, users)
	return *users, err
}

func (c *Client) ListProjects(opt *Filter) ([]Project, error) {
	projects := new([]Project)
	err := c.Request("projects", opt, projects)
	return *projects, err
}

func (c *Client) ListTasks(opt *Filter) ([]Task, error) {
	tasks := new([]Task)
	err := c.Request("tasks", opt, tasks)
	return *tasks, err
}

func (c *Client) ListTags(opt *Filter) ([]Tag, error) {
	tags := new([]Tag)
	err := c.Request("tags", opt, tags)
	return *tags, err
}

func (c *Client) Request(path string, opt *Filter, v interface{}) error {
	if opt == nil {
		opt = &Filter{}
	}
	if opt.OptFields == "" {
		// default options should not modify provided options
		newOpt := *opt
		opt = &newOpt
		opt.OptFields = defaultOptFields[path]
	}
	urlStr, err := addOptions(path, opt)
	if err != nil {
		return err
	}
	rel, err := url.Parse(urlStr)
	if err != nil {
		return err
	}
	u := c.BaseURL.ResolveReference(rel)
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return err
	}

	req.Header.Add("User-Agent", c.UserAgent)
	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	res := &Response{Data: v}
	err = json.NewDecoder(resp.Body).Decode(res)
	if len(res.Errors) > 0 {
		return res.Errors[0]
	}
	return err
}

func addOptions(s string, opt interface{}) (string, error) {
	v := reflect.ValueOf(opt)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return s, nil
	}
	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}
	qs, err := query.Values(opt)
	if err != nil {
		return s, err
	}
	u.RawQuery = qs.Encode()
	return u.String(), nil
}
