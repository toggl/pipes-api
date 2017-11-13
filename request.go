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
	"strings"
	"time"

	"github.com/bugsnag/bugsnag-go"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	gouuid "github.com/nu7hatch/gouuid"
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
		requestStarted := time.Now()

		uuidToken := uuid(r)

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

		req := Request{w, r, body}
		resp := handler(req)

		defer func() {
			log.Println(uuidToken, r.Method, resp.status, r.URL, "-", time.Since(requestStarted))
		}()

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

func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer context.Clear(r)

	u4, err := gouuid.NewV4()
	if err != nil {
		log.Print(err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	uuidToken := u4.String()
	context.Set(r, uuidKey, uuidToken)

	log.Println(uuidToken, "Start", r.Method, r.URL, "for", parseRemoteAddr(r))

	w.Header().Set("Cache-Control", "no-cache, private, no-store, must-revalidate, max-stale=0, post-check=0, pre-check=0")

	if origin, ok := isWhiteListedCorsOrigin(r); ok {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Depth, User-Agent, X-File-Size, X-Requested-With, If-Modified-Since, X-File-Name, Cache-Control, Authorization, Accept, Accept-Encoding, Accept-Language, Access-Control-Request-Headers, Access-Control-Request-Method, Connection, Host, Origin, User-Agent")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS, PUT, POST, DELETE")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Max-Age", "1728000")
	}

	if r.Method == "OPTIONS" {
		w.Header().Set("Content-Length", "0")
		w.WriteHeader(http.StatusOK)
		return
	}
	router.Routes.ServeHTTP(w, r)
}
