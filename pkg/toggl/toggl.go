package toggl

// Workspace represents toggl workspace
type Workspace struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type WorkspaceResponse struct {
	Workspace *Workspace `json:"data"`
}

// Account represents account from third party integration
type Account struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type User struct {
	ID             int    `json:"id,omitempty"`
	Email          string `json:"email"`
	Name           string `json:"name"`
	SendInvitation bool   `json:"send_invitation,omitempty"`

	// ForeignID is a meta information which won't be saved into DB on "toggl_api" side.
	ForeignID string `json:"foreign_id,omitempty"`
}

type Client struct {
	ID   int    `json:"id,omitempty"`
	Name string `json:"name"`

	// ForeignID is a meta information which won't be saved into DB on "toggl_api" side.
	ForeignID string `json:"foreign_id,omitempty"`
}

type ClientRequest struct {
	Clients []*Client `json:"clients"`
}

type ClientsImport struct {
	Clients       []*Client `json:"clients"`
	Notifications []string  `json:"notifications"`
}

func (p *ClientsImport) Count() int {
	return len(p.Clients)
}

type Project struct {
	ID       int    `json:"id,omitempty"`
	Name     string `json:"name,omitempty"`
	Active   bool   `json:"active,omitempty"`
	Billable bool   `json:"billable,omitempty"`
	ClientID int    `json:"cid,omitempty"`

	// ForeignID is a meta information which won't be saved into DB on "toggl_api" side.
	ForeignID       string `json:"foreign_id,omitempty"`
	ForeignClientID string `json:"-"`
}

type Task struct {
	ID        int    `json:"id,omitempty"`
	Name      string `json:"name"`
	Active    bool   `json:"active"`
	ProjectID int    `json:"pid"`

	// ForeignID is a meta information which won't be saved into DB on "toggl_api" side.
	ForeignID        string `json:"foreign_id,omitempty"`
	ForeignProjectID string `json:"-"`
}

type TimeEntry struct {
	ID                int    `json:"id"`
	ProjectID         int    `json:"pid,omitempty"`
	TaskID            int    `json:"tid,omitempty"`
	UserID            int    `json:"uid,omitempty"`
	Billable          bool   `json:"billable"`
	Start             string `json:"start"`
	Stop              string `json:"stop,omitempty"`
	DurationInSeconds int    `json:"duration"`
	Description       string `json:"description,omitempty"`

	// ForeignID is a meta information which won't be saved into DB on "toggl_api" side.
	ForeignID        string `json:"foreign_id,omitempty"`
	ForeignTaskID    string `json:"-"`
	ForeignUserID    string `json:"-"`
	ForeignProjectID string `json:"-"`
}

type AccountsResponse struct {
	Error    string     `json:"error"`
	Accounts []*Account `json:"accounts"`
}

type UsersRequest struct {
	Users []*User `json:"users"`
}

type UsersImport struct {
	WorkspaceUsers []*User  `json:"users"`
	Notifications  []string `json:"notifications"`
}

func (p *UsersImport) Count() int {
	return len(p.WorkspaceUsers)
}

type UsersResponse struct {
	Error string  `json:"error"`
	Users []*User `json:"users"`
}

type ClientsResponse struct {
	Error   string    `json:"error"`
	Clients []*Client `json:"clients"`
}

type ProjectsResponse struct {
	Error    string     `json:"error"`
	Projects []*Project `json:"projects"`
}

type ProjectRequest struct {
	Projects []*Project `json:"projects"`
}

type ProjectsImport struct {
	Projects      []*Project `json:"projects"`
	Notifications []string   `json:"notifications"`
}

func (p *ProjectsImport) Count() int {
	return len(p.Projects)
}

type TaskRequest struct {
	Tasks []*Task `json:"tasks"`
}
type TasksImport struct {
	Tasks         []*Task  `json:"tasks"`
	Notifications []string `json:"notifications"`
}

func (p *TasksImport) Count() int {
	return len(p.Tasks)
}

type TasksResponse struct {
	Error string  `json:"error"`
	Tasks []*Task `json:"tasks"`
}
