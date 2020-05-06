package main

type (
	// Workspace represents toggl workspace
	Workspace struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	// Account represents account from third party integration
	Account struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}

	User struct {
		ID             int    `json:"id,omitempty"`
		Email          string `json:"email"`
		Name           string `json:"name"`
		SendInvitation bool   `json:"send_invitation,omitempty"`
		ForeignID      string `json:"foreign_id,omitempty"`
	}

	Client struct {
		ID        int    `json:"id,omitempty"`
		Name      string `json:"name"`
		ForeignID string `json:"foreign_id,omitempty"`
	}

	Project struct {
		ID       int    `json:"id,omitempty"`
		Name     string `json:"name,omitempty"`
		Active   bool   `json:"active,omitempty"`
		Billable bool   `json:"billable,omitempty"`
		ClientID int    `json:"cid,omitempty"`

		ForeignID       string `json:"foreign_id,omitempty"`
		foreignClientID string
	}

	Task struct {
		ID        int    `json:"id,omitempty"`
		Name      string `json:"name"`
		Active    bool   `json:"active"`
		ProjectID int    `json:"pid"`

		ForeignID        string `json:"foreign_id,omitempty"`
		foreignProjectID string
	}

	TimeEntry struct {
		ID                int    `json:"id"`
		ProjectID         int    `json:"pid,omitempty"`
		TaskID            int    `json:"tid,omitempty"`
		UserID            int    `json:"uid,omitempty"`
		Billable          bool   `json:"billable"`
		Start             string `json:"start"`
		Stop              string `json:"stop,omitempty"`
		DurationInSeconds int    `json:"duration"`
		Description       string `json:"description,omitempty"`

		foreignID        string
		foreignTaskID    string
		foreignUserID    string
		foreignProjectID string
	}

	AccountsResponse struct {
		Error    string     `json:"error"`
		Accounts []*Account `json:"accounts"`
	}

	UsersResponse struct {
		Error string  `json:"error"`
		Users []*User `json:"users"`
	}

	ClientsResponse struct {
		Error   string    `json:"error"`
		Clients []*Client `json:"clients"`
	}

	ProjectsResponse struct {
		Error    string     `json:"error"`
		SupportsClient bool `json:"supports_client"`
		Projects []*Project `json:"projects"`
	}

	TasksResponse struct {
		Error string  `json:"error"`
		Tasks []*Task `json:"tasks"`
	}
)
