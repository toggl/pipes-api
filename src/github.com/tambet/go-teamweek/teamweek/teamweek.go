package teamweek

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

const (
	libraryVersion = "0.1"
	userAgent      = "go-teamweek/" + libraryVersion
	defaultBaseURL = "https://new.teamweek.com/api/v3/"
)

type (
	Client struct {
		client    *http.Client
		BaseURL   *url.URL
		UserAgent string
	}

	Account struct {
		ID          int64   `json:"id,omitempty"`
		Name        string  `json:"name,omitempty"`
		IsDemo      bool    `json:"is_demo,omitempty"`
		SuspendedAt string  `json:"suspended_at,omitempty"`
		Groups      []Group `json:"groups,omitempty"`
	}

	userFields struct {
		ID         int64  `json:"id,omitempty"`
		Email      string `json:"email,omitempty"`
		Name       string `json:"name,omitempty"`
		Initials   string `json:"initials,omitempty"`
		PictureUrl string `json:"picture_url,omitempty"`
		HasPicture bool   `json:"has_picture,omitempty"`
		Color      string `json:"color,omitempty"`
	}

	Profile struct {
		userFields
		Accounts    []Account     `json:"accounts,omitempty"`
		Invitations []interface{} `json:"invitations,omitempty"`
		CreatedAt   string        `json:"created_at,omitempty"`
		UpdatedAt   string        `json:"updated_at,omitempty"`
	}

	User struct {
		userFields
		Weight int64 `json:"weight,omitempty"`
		Dummy  bool  `json:"dummy,omitempty"`
	}

	Project struct {
		ID    int64  `json:"id,omitempty"`
		Name  string `json:"name,omitempty"`
		Color string `json:"color,omitempty"`
	}

	Task struct {
		ID             int64    `json:"id,omitempty"`
		Name           string   `json:"name,omitempty"`
		StartDate      string   `json:"start_date,omitempty"`
		EndDate        string   `json:"end_date,omitempty"`
		StartTime      string   `json:"start_time,omitempty"`
		EndTime        string   `json:"end_time,omitempty"`
		Color          string   `json:"color,omitempty"`
		EstimatedHours float64  `json:"estimated_hours,omitempty"`
		Pinned         bool     `json:"pinned,omitempty"`
		Done           bool     `json:"done,omitempty"`
		UserID         int64    `json:"user_id,omitempty"`
		Project        *Project `json:"project,omitempty"`
	}

	Milestone struct {
		ID      int64  `json:"id,omitempty"`
		Name    string `json:"name,omitempty"`
		Date    string `json:"date,omitempty"`
		Done    bool   `json:"done,omitempty"`
		Holiday bool   `json:"holiday,omitempty"`
	}

	Group struct {
		ID         int64        `json:"id,omitempty"`
		Name       string       `json:"name,omitempty"`
		AccountID  int64        `json:"account_id,omitempty"`
		Membership []Membership `json:"memberships,omitempty"`
	}

	Membership struct {
		ID      int64 `json:"id,omitempty"`
		GroupID int64 `json:"group_id,omitempty"`
		UserID  int64 `json:"user_id,omitempty"`
		Weight  int64 `json:"weight,omitempty"`
	}
)

func (c *Client) Profile() (*Profile, error) {
	profile := new(Profile)
	err := c.Request("me.json", profile)
	return profile, err
}

func (c *Client) ListAccounts() ([]Account, error) {
	accounts := new([]Account)
	err := c.Request("me/accounts.json", accounts)
	return *accounts, err
}

func (c *Client) ListAccountUsers(accountID int64) ([]User, error) {
	users := new([]User)
	err := c.Request(fmt.Sprintf("%d/users.json", accountID), users)
	return *users, err
}

func (c *Client) ListAccountProjects(accountID int64) ([]Project, error) {
	projects := new([]Project)
	err := c.Request(fmt.Sprintf("%d/projects.json", accountID), projects)
	return *projects, err
}

func (c *Client) ListAccountMilestones(accountID int64) ([]Milestone, error) {
	milestones := new([]Milestone)
	err := c.Request(fmt.Sprintf("%d/milestones.json", accountID), milestones)
	return *milestones, err
}

func (c *Client) ListAccountGroups(accountID int64) ([]Group, error) {
	groups := new([]Group)
	err := c.Request(fmt.Sprintf("%d/groups.json", accountID), groups)
	return *groups, err
}

func (c *Client) ListAccountTasks(accountID int64) ([]Task, error) {
	tasks := new([]Task)
	err := c.Request(fmt.Sprintf("%d/tasks.json", accountID), tasks)
	return *tasks, err
}

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	baseURL, _ := url.Parse(defaultBaseURL)
	client := &Client{client: httpClient, BaseURL: baseURL, UserAgent: userAgent}
	return client
}

func (c *Client) Request(urlStr string, v interface{}) error {
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

	return json.NewDecoder(resp.Body).Decode(v)
}
