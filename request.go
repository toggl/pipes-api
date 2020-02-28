package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/bugsnag/bugsnag-go"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
)

type (
	Response struct {
		status      int
		content     interface{}
		contentType string
	}

	Request struct {
		w    http.ResponseWriter
		r    *http.Request
		body []byte
	}

	HandlerFunc func(req Request) Response
)

// Request context keys
type key int

const (
	uuidKey           key = 0
	workspaceIDKey    key = 1
	workspaceTokenKey key = 2
	serviceIDKey      key = 3
	pipeIDKey         key = 4
)

func badRequest(explanation interface{}) Response {
	if s, isString := explanation.(string); isString {
		return Response{http.StatusBadRequest, errors.New(s), "application/json"}
	}
	return Response{http.StatusBadRequest, explanation.(error), "application/json"}
}

func internalServerError(err string) Response {
	return Response{http.StatusInternalServerError, err, "application/json"}
}

func ok(content interface{}) Response {
	return Response{http.StatusOK, content, "application/json"}
}

func accepted(content interface{}) Response {
	return Response{http.StatusAccepted, content, "application/json"}
}

func found(location string) Response {
	return Response{http.StatusFound, location, "application/json"}
}

func noContent() Response {
	return Response{http.StatusNoContent, nil, "application/json"}
}

func badGateway(err string) Response {
	return Response{http.StatusBadGateway, err, "application/json"}
}

func serviceUnavailable(reasons interface{}) Response {
	return Response{http.StatusServiceUnavailable, reasons, "application/json"}
}

func (req Request) redirectWithError(err string) Response {
	return found(urls.ReturnURL[environment] + "?err=" + url.QueryEscape(err))
}

func uuid(r *http.Request) string {
	return fmt.Sprintf("%v", context.Get(r, uuidKey))
}

func currentWorkspaceID(r *http.Request) int {
	if v, ok := context.GetOk(r, workspaceIDKey); ok {
		return v.(int)
	}
	return 0
}

func currentServicePipeID(r *http.Request) (string, string) {
	var serviceID, pipeID string
	if v, ok := context.GetOk(r, serviceIDKey); ok {
		serviceID = v.(string)
	}
	if v, ok := context.GetOk(r, pipeIDKey); ok {
		pipeID = v.(string)
	}
	return serviceID, pipeID
}

func currentWorkspaceToken(r *http.Request) string {
	if v, ok := context.GetOk(r, workspaceTokenKey); ok {
		return v.(string)
	}
	return ""
}

func parseRemoteAddr(r *http.Request) string {
	if forwarded := r.Header.Get("X-forwarded-for"); forwarded != "" {
		return forwarded
	}
	return r.RemoteAddr
}

func withService(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serviceID := mux.Vars(r)["service"]
		if !serviceType.MatchString(serviceID) {
			http.Error(w, "Missing or invalid service", http.StatusBadRequest)
			return
		}
		pipeID := mux.Vars(r)["pipe"]
		if !pipeType.MatchString(pipeID) {
			http.Error(w, "Missing or invalid pipe", http.StatusBadRequest)
			return
		}
		context.Set(r, serviceIDKey, serviceID)
		context.Set(r, pipeIDKey, pipeID)
		handler(w, r)
	}
}

func withAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authData, err := parseToken(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if authData == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		var workspaceID int
		workspaceID, err = getTogglWorkspaceID(authData.Username)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		context.Set(r, workspaceIDKey, workspaceID)
		context.Set(r, workspaceTokenKey, authData.Username)
		handler(w, r)
	}
}

// handleRequest wraps API request/response calls and writes the response out.
func handleRequest(handler HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// take care of panic
		defer func() {
			if recover() != nil {
				log.Println("panic when handling", r.URL.Path)
				bugsnag.Recover(bugsnag.StartSession(r.Context()))
			}
		}()

		uuidToken := uuid(r)
		requestStarted := time.Now()

		// define resp so it can be used in log
		var resp Response

		// log request
		log.Println(uuidToken, "Start", r.Method, r.URL, "for", parseRemoteAddr(r))
		defer func() {
			log.Println(uuidToken, r.Method, resp.status, r.URL, "-", time.Since(requestStarted))
		}()

		// Parse request body, if any
		var body []byte
		if r.Body != nil {
			defer r.Body.Close()
			b, err := ioutil.ReadAll(r.Body)
			if err != nil {
				log.Println(uuidToken, "Error:", err, r)
				bugsnag.Notify(err, r)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			body = b
			if len(body) > 0 {
				log.Println(uuidToken, "Input:", string(body))
			}
		}

		// run the actual handler
		req := Request{w, r, body}
		resp = handler(req)

		// Handle error
		if err, isError := resp.content.(error); isError {
			log.Println(uuidToken, "Error:", err, r)
			if resp.status < 400 || resp.status >= 500 {
				go bugsnag.Notify(err,
					bugsnag.MetaData{
						"request": {
							"uuid": uuidToken,
						},
					})
			}
			http.Error(w, err.Error(), resp.status)
			return
		}

		// Handle redirect
		if resp.status == http.StatusFound {
			location := resp.content.(string)
			log.Println(uuidToken, "Redirect:", location)
			http.Redirect(w, r, location, resp.status)
			return
		}

		// Encode JSON response
		var output []byte
		if resp.contentType == "application/json" {
			b, err := json.Marshal(resp.content)
			if err != nil {
				log.Print(err.Error())
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			output = b
		} else if resp.content != nil {
			output = []byte(fmt.Sprintf("%v", resp.content))
		}

		// Log output, except for GET results, which tend to be spammy.
		if r.Method != "GET" {
			log.Println(uuidToken, "Output", resp.contentType, string(output))
		}

		// Write output
		w.Header().Set("Content-Length", strconv.Itoa(len(output)))
		w.Header().Set("Content-type", resp.contentType)
		w.WriteHeader(resp.status)
		w.Write(output)
	}
}
