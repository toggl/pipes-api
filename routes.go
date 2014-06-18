package main

import (
	"github.com/gorilla/mux"
	"net/http"
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

	v1.HandleFunc("/integrations/{service}/accounts", withAuth(handleRequest(getServiceAccounts))).Methods("GET")
	v1.HandleFunc("/integrations/{service}/auth_url", withAuth(handleRequest(getAuthURL))).Methods("GET")
	v1.HandleFunc("/integrations/{service}/authorizations", withAuth(handleRequest(postAuthorization))).Methods("POST")
	v1.HandleFunc("/integrations/{service}/authorizations", withAuth(handleRequest(deleteAuthorization))).Methods("DELETE")

	v1.HandleFunc("/integrations/{service}/pipes/{pipe}/users", withAuth(handleRequest(getServiceUsers))).Methods("GET")
	v1.HandleFunc("/integrations/{service}/pipes/{pipe}/run", withService(withAuth(handleRequest(postPipeRun)))).Methods("POST")

	http.Handle("/", routes)
}
