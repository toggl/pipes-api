package teamweek

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

var (
	client *Client
	mux    *http.ServeMux
	server *httptest.Server
)

func setup() {
	client = NewClient(nil)
	mux = http.NewServeMux()
	server = httptest.NewServer(mux)
	url, _ := url.Parse(server.URL)
	client.BaseURL = url
}

func teardown() {
	server.Close()
}

func TestNewClient(t *testing.T) {
	c := NewClient(nil)

	if c.BaseURL.String() != defaultBaseURL {
		t.Errorf("NewClient BaseURL = %v, want %v", c.BaseURL.String(), defaultBaseURL)
	}
	if c.UserAgent != userAgent {
		t.Errorf("NewClient UserAgent = %v, want %v", c.UserAgent, userAgent)
	}
}

func TestUserProfile(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/me", func(w http.ResponseWriter, r *http.Request) {
		js := `{
			"workspaces": [{
				"updated_at": "2018-09-14T07:51:26.463375",
				"suspended_at": null,
				"role": "admin",
				"name": "Choucroute",
				"id": 42,
				"custom_colors": [],
				"created_at": "2018-09-14T07:51:26.463362",
				"active": true
			}],
			"updated_at": "2018-09-14T07:55:47.397289",
			"project_ids": [],
			"picture_url": null,
			"name": "Fred",
			"manager": true,
			"invitations": [],
			"initials": "fr",
			"id": 123456,
			"has_picture": false,
			"email": "frederic@toggl.com",
			"created_at": "2018-09-14T07:50:22.036305",
			"color_id": 22}`
		fmt.Fprint(w, js)
	})

	profile, err := client.GetUserProfile()
	if err != nil {
		t.Errorf("UserProfile returned error: %v", err)
	}

	want := &UserProfile{
		ID:       123456,
		Name:     "Fred",
		Email:    "frederic@toggl.com",
		ColorID:  22,
		Initials: "fr",
		Workspaces: []Workspace{
			{
				ID:        42,
				Name:      "Choucroute",
				Active:    true,
				Role:      "admin",
				CreatedAt: "2018-09-14T07:51:26.463362",
				UpdatedAt: "2018-09-14T07:51:26.463375"}},
		CreatedAt: "2018-09-14T07:50:22.036305",
		UpdatedAt: "2018-09-14T07:55:47.397289",
	}
	if !reflect.DeepEqual(profile, want) {
		t.Errorf("UserProfile returned %+v, want %+v", profile, want)
	}
}

func TestWorkspaceMembers(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/1/members", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[
			{"id":1,"email":"test1@teamweek.com"},
			{"id":2,"email":"test2@teamweek.com"}
		]`)
	})

	users, err := client.ListWorkspaceMembers(1)
	if err != nil {
		t.Errorf("ListWorkspaceMembers returned error: %v", err)
	}

	want := []Member{
		{ID: 1, Email: "test1@teamweek.com"},
		{ID: 2, Email: "test2@teamweek.com"},
	}

	if !reflect.DeepEqual(users, want) {
		t.Errorf("ListWorkspaceMembers returned %+v, want %+v", users, want)
	}
}

func TestListWorkspaceProjects(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/1/projects", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[
			{"id":1,"name":"Showtime"},
			{"id":2,"name":"Quality time"}
		]`)
	})

	projects, err := client.ListWorkspaceProjects(1)
	if err != nil {
		t.Errorf("ListWorkspaceProjects returned error: %v", err)
	}

	want := []Project{
		{ID: 1, Name: "Showtime"},
		{ID: 2, Name: "Quality time"},
	}

	if !reflect.DeepEqual(projects, want) {
		t.Errorf("ListWorkspaceProjects returned %+v, want %+v", projects, want)
	}
}

func TestListWorkspaceMilestones(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/1/milestones", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[
			{"id":1,"name":"End of season 1"},
			{"id":2,"name":"End of season 2"}
		]`)
	})

	milestones, err := client.ListWorkspaceMilestones(1)
	if err != nil {
		t.Errorf("ListWorkspaceMilestones returned error: %v", err)
	}

	want := []Milestone{
		{ID: 1, Name: "End of season 1"},
		{ID: 2, Name: "End of season 2"},
	}

	if !reflect.DeepEqual(milestones, want) {
		t.Errorf("ListWorkspaceMilestones returned %+v, want %+v", milestones, want)
	}
}

func TestListWorkspaceGroups(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/1/groups", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[
			{"id":1,"name":"Red Muppets"},
			{"id":2,"name":"Blue Muppets"}
		]`)
	})

	groups, err := client.ListWorkspaceGroups(1)
	if err != nil {
		t.Errorf("ListWorkspaceGroups returned error: %v", err)
	}

	want := []Group{
		{ID: 1, Name: "Red Muppets"},
		{ID: 2, Name: "Blue Muppets"},
	}

	if !reflect.DeepEqual(groups, want) {
		t.Errorf("ListWorkspaceGroups returned %+v, want %+v", groups, want)
	}
}

func TestListWorkspaceTasks(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/1/tasks", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[
			{"id":1,"name":"Act like muppet"},
			{"id":2,"name":"Lunch with Abby"}
		]`)
	})

	tasks, err := client.ListWorkspaceTasks(1)
	if err != nil {
		t.Errorf("ListWorkspaceTasks returned error: %v", err)
	}

	want := []Task{
		{ID: 1, Name: "Act like muppet"},
		{ID: 2, Name: "Lunch with Abby"},
	}

	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("ListWorkspaceTasks returned %+v, want %+v", tasks, want)
	}
}
