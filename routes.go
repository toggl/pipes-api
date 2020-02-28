
package main

import (
	"log"
	"net/http"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	gouuid "github.com/nu7hatch/gouuid"
)

type Router struct {
	Routes *mux.Router
}

var routes *Router

func init() {
	routes = &Router{Routes: mux.NewRouter()}

	v1 := routes.Routes.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/status", handleRequest(getStatus)).Methods("GET")
	v1.HandleFunc("/integrations", withAuth(handleRequest(getIntegrations))).Methods("GET")

	v1.HandleFunc("/integrations/{service}/pipes/{pipe}", withAuth(handleRequest(getIntegrationPipe))).Methods("GET")
	v1.HandleFunc("/integrations/{service}/pipes/{pipe}/setup", withAuth(handleRequest(putPipeSetup))).Methods("PUT")
	v1.HandleFunc("/integrations/{service}/pipes/{pipe}/setup", withAuth(handleRequest(postPipeSetup))).Methods("POST")
	v1.HandleFunc("/integrations/{service}/pipes/{pipe}/setup", withAuth(handleRequest(deletePipeSetup))).Methods("DELETE")
	v1.HandleFunc("/integrations/{service}/pipes/{pipe}/log", withService(withAuth(handleRequest(getServicePipeLog)))).Methods("GET")
	v1.HandleFunc("/integrations/{service}/pipes/{pipe}/clear_connections", withService(withAuth(handleRequest(postServicePipeClearConnections)))).Methods("POST")

	v1.HandleFunc("/integrations/{service}/accounts", withAuth(handleRequest(getServiceAccounts))).Methods("GET")
	v1.HandleFunc("/integrations/{service}/auth_url", withAuth(handleRequest(getAuthURL))).Methods("GET")
	v1.HandleFunc("/integrations/{service}/authorizations", withAuth(handleRequest(postAuthorization))).Methods("POST")
	v1.HandleFunc("/integrations/{service}/authorizations", withAuth(handleRequest(deleteAuthorization))).Methods("DELETE")

	v1.HandleFunc("/integrations/{service}/pipes/{pipe}/users", withAuth(handleRequest(getServiceUsers))).Methods("GET")
	v1.HandleFunc("/integrations/{service}/pipes/{pipe}/run", withService(withAuth(handleRequest(postPipeRun)))).Methods("POST")

	http.Handle("/", routes)
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
