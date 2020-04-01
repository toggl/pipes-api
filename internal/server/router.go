package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/bugsnag/bugsnag-go"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
)

type Router struct {
	Routes        *mux.Router
	CorsWhiteList []string

	mx sync.Mutex
}

func NewRouter(corsWhiteList []string) *Router {
	routes := &Router{
		Routes:        mux.NewRouter(),
		CorsWhiteList: corsWhiteList,
	}
	return routes
}

func (router *Router) AttachHandlers(c *Controller, mw *Middleware) *Router {

	v1 := router.Routes.PathPrefix("/api/v1").Subrouter()
	v1.HandleFunc("/status", mw.withUUID(handleRequest(c.GetStatusHandler))).Methods("GET")
	v1.HandleFunc("/integrations", mw.withUUID(mw.withAuth(handleRequest(c.GetIntegrationsHandler)))).Methods("GET")

	v1.HandleFunc("/integrations/{service}/pipes/{pipe}", mw.withUUID(mw.withAuth(handleRequest(c.ReadPipeHandler)))).Methods("GET")
	v1.HandleFunc("/integrations/{service}/pipes/{pipe}/setup", mw.withUUID(mw.withAuth(handleRequest(c.UpdatePipeHandler)))).Methods("PUT")
	v1.HandleFunc("/integrations/{service}/pipes/{pipe}/setup", mw.withUUID(mw.withAuth(handleRequest(c.CreatePipeHandler)))).Methods("POST")
	v1.HandleFunc("/integrations/{service}/pipes/{pipe}/setup", mw.withUUID(mw.withAuth(handleRequest(c.DeletePipeHandler)))).Methods("DELETE")
	v1.HandleFunc("/integrations/{service}/pipes/{pipe}/log", mw.withUUID(mw.withService(mw.withAuth(handleRequest(c.GetServicePipeLogHandler))))).Methods("GET")
	v1.HandleFunc("/integrations/{service}/pipes/{pipe}/clear_connections", mw.withUUID(mw.withService(mw.withAuth(handleRequest(c.PostServicePipeClearIDMappingsHandler))))).Methods("POST") // TODO: Remove (Probably dead endpoint).
	v1.HandleFunc("/integrations/{service}/pipes/{pipe}/users", mw.withUUID(mw.withAuth(handleRequest(c.GetServiceUsersHandler)))).Methods("GET")
	v1.HandleFunc("/integrations/{service}/pipes/{pipe}/run", mw.withAuth(mw.withService(mw.withUUID(handleRequest(c.PostPipeRunHandler))))).Methods("POST")

	v1.HandleFunc("/integrations/{service}/accounts", mw.withUUID(mw.withAuth(handleRequest(c.GetServiceAccountsHandler)))).Methods("GET")
	v1.HandleFunc("/integrations/{service}/auth_url", mw.withUUID(mw.withAuth(handleRequest(c.GetAuthURLHandler)))).Methods("GET")
	v1.HandleFunc("/integrations/{service}/authorizations", mw.withUUID(mw.withAuth(handleRequest(c.CreateAuthorizationHandler)))).Methods("POST")
	v1.HandleFunc("/integrations/{service}/authorizations", mw.withUUID(mw.withAuth(handleRequest(c.DeleteAuthorizationHandler)))).Methods("DELETE")

	return router
}

func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache, private, no-store, must-revalidate, max-stale=0, post-check=0, pre-check=0")

	if origin, ok := router.isWhiteListedCorsOrigin(r); ok {
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

func (router *Router) isWhiteListedCorsOrigin(r *http.Request) (string, bool) {
	router.mx.Lock()
	defer router.mx.Unlock()

	origin := r.Header.Get("Origin")
	for _, s := range router.CorsWhiteList {
		if s == origin {
			return origin, true
		}
	}
	return "", false
}

func Start(port int, routes *Router) {
	http.Handle("/", routes)

	listenAddress := fmt.Sprintf(":%d", port)
	log.Printf("pipes (PID: %d) is starting on %s\n=> Ctrl-C to shutdown server\n", os.Getpid(), listenAddress)
	log.Fatal(http.ListenAndServe(listenAddress, bugsnag.Handler(context.ClearHandler(http.DefaultServeMux))))
}
