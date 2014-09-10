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

func TestListAccounts(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/me/accounts.json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[
 			{"id":1,"name":"Account 1","is_demo":false},
 			{"id":2,"name":"Account 2","is_demo":true}
		]`)
	})

	accounts, err := client.ListAccounts()
	if err != nil {
		t.Errorf("ListAccounts returned error: %v", err)
	}

	want := []Account{
		{ID: 1, Name: "Account 1"},
		{ID: 2, Name: "Account 2", IsDemo: true},
	}

	if !reflect.DeepEqual(accounts, want) {
		t.Errorf("ListAccounts returned %+v, want %+v", accounts, want)
	}
}

func TestProfile(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/me.json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"id":1,"email":"test@teamweek.com"}`)
	})

	profile, err := client.Profile()
	if err != nil {
		t.Errorf("Profile returned error: %v", err)
	}

	want := &Profile{ID: 1, Email: "test@teamweek.com"}

	if !reflect.DeepEqual(profile, want) {
		t.Errorf("Profile returned %+v, want %+v", profile, want)
	}
}

func TestListAccountUsers(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/1/users.json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[
			{"id":1,"email":"test1@teamweek.com"},
			{"id":2,"email":"test2@teamweek.com"}
		]`)
	})

	users, err := client.ListAccountUsers(1)
	if err != nil {
		t.Errorf("ListAccountUsers returned error: %v", err)
	}

	want := []User{
		{ID: 1, Email: "test1@teamweek.com"},
		{ID: 2, Email: "test2@teamweek.com"},
	}

	if !reflect.DeepEqual(users, want) {
		t.Errorf("ListAccountUsers returned %+v, want %+v", users, want)
	}
}

func TestListAccountProjects(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/1/projects.json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[
			{"id":1,"name":"Showtime"},
			{"id":2,"name":"Quality time"}
		]`)
	})

	projects, err := client.ListAccountProjects(1)
	if err != nil {
		t.Errorf("ListAccountProjects returned error: %v", err)
	}

	want := []Project{
		{ID: 1, Name: "Showtime"},
		{ID: 2, Name: "Quality time"},
	}

	if !reflect.DeepEqual(projects, want) {
		t.Errorf("ListAccountProjects returned %+v, want %+v", projects, want)
	}
}

func TestListAccountMilestones(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/1/milestones.json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[
			{"id":1,"name":"End of season 1"},
			{"id":2,"name":"End of season 2"}
		]`)
	})

	milestones, err := client.ListAccountMilestones(1)
	if err != nil {
		t.Errorf("ListAccountMilestones returned error: %v", err)
	}

	want := []Milestone{
		{ID: 1, Name: "End of season 1"},
		{ID: 2, Name: "End of season 2"},
	}

	if !reflect.DeepEqual(milestones, want) {
		t.Errorf("ListAccountMilestones returned %+v, want %+v", milestones, want)
	}
}

func TestListAccountGroups(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/1/groups.json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[
			{"id":1,"name":"Red Muppets"},
			{"id":2,"name":"Blue Muppets"}
		]`)
	})

	groups, err := client.ListAccountGroups(1)
	if err != nil {
		t.Errorf("ListAccountGroups returned error: %v", err)
	}

	want := []Group{
		{ID: 1, Name: "Red Muppets"},
		{ID: 2, Name: "Blue Muppets"},
	}

	if !reflect.DeepEqual(groups, want) {
		t.Errorf("ListAccountGroups returned %+v, want %+v", groups, want)
	}
}

func TestListAccountTasks(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/1/tasks.json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `[
			{"id":1,"name":"Act like muppet"},
			{"id":2,"name":"Lunch with Abby"}
		]`)
	})

	tasks, err := client.ListAccountTasks(1)
	if err != nil {
		t.Errorf("ListAccountTasks returned error: %v", err)
	}

	want := []Task{
		{ID: 1, Name: "Act like muppet"},
		{ID: 2, Name: "Lunch with Abby"},
	}

	if !reflect.DeepEqual(tasks, want) {
		t.Errorf("ListAccountTasks returned %+v, want %+v", tasks, want)
	}
}

func TestInvalidURL(t *testing.T) {
	client = NewClient(nil)
	err := client.Request("/%s/error", nil)
	if err == nil {
		t.Errorf("Expected 'invalid URL escape' error")
	}
}

func TestHandleHttpError(t *testing.T) {
	setup()
	defer teardown()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Bad Request", 400)
	})
	err := client.Request("/", nil)
	if err == nil {
		t.Errorf("Expected 'Bad Request' error")
	}
}

func TestInvalidNewRequest(t *testing.T) {
	client = NewClient(nil)
	client.BaseURL = &url.URL{Host: "%s"}
	err := client.Request("/", nil)

	if err == nil {
		t.Error("Expected error to be returned.")
	}
	if err, ok := err.(*url.Error); !ok {
		t.Errorf("Expected a URL error; got %#v.", err)
	}
}

func TestHttpClientError(t *testing.T) {
	client = NewClient(nil)
	client.BaseURL = &url.URL{}
	err := client.Request("/", nil)

	if err == nil {
		t.Error("Expected error to be returned.")
	}
	if err, ok := err.(*url.Error); !ok {
		t.Errorf("Expected a URL error; got %#v.", err)
	}
}
