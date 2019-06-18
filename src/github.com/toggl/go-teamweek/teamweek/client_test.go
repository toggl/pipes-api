package teamweek

import (
	"net/http"
	"net/url"
	"testing"
)

func TestInvalidURL(t *testing.T) {
	client = NewClient(nil)
	err := client.get("/%s/error", nil)
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
	err := client.get("/", nil)
	if err == nil {
		t.Errorf("Expected 'Bad Request' error")
	}
}

func TestInvalidNewRequest(t *testing.T) {
	client = NewClient(nil)
	client.BaseURL = &url.URL{Host: "%s"}
	err := client.get("/", nil)

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
	err := client.get("/", nil)

	if err == nil {
		t.Error("Expected error to be returned.")
	}
	if err, ok := err.(*url.Error); !ok {
		t.Errorf("Expected a URL error; got %#v.", err)
	}
}

func TestServiceInternalError(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", 500)
	})

	if err := client.get("/", nil); err == nil {
		t.Errorf("Expected 'Internal Server Error'")
	}
}

func TestUnauthorizedError(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Unauthorized", 401)
	})

	if err := client.get("/", nil); err == nil {
		t.Errorf("Expected 'Unauthorized'")
	}
}

func TestUnexpectedStatus(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Accepted", 442)
	})

	if err := client.get("/", nil); err == nil {
		t.Errorf("Unexpected status code error")
	}
}
