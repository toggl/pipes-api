package asana

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

func TestListWorkspaces(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/workspaces", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"data":[
 			{"id":1,"name":"Organization 1"},
 			{"id":2,"name":"Organization 2"}
		]}`)
	})

	workspaces, err := client.ListWorkspaces()
	if err != nil {
		t.Errorf("ListWorkspaces returned error: %v", err)
	}

	want := []Workspace{
		{ID: 1, Name: "Organization 1"},
		{ID: 2, Name: "Organization 2"},
	}

	if !reflect.DeepEqual(workspaces, want) {
		t.Errorf("ListWorkspaces returned %+v, want %+v", workspaces, want)
	}
}
