package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	gouuid "github.com/nu7hatch/gouuid"
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
	v1.HandleFunc("/status", handleRequest(c.GetStatus)).Methods("GET")
	v1.HandleFunc("/integrations", mw.withAuth(handleRequest(c.GetIntegrations))).Methods("GET")

	v1.HandleFunc("/integrations/{service}/pipes/{pipe}", mw.withAuth(handleRequest(c.GetIntegrationPipe))).Methods("GET")
	v1.HandleFunc("/integrations/{service}/pipes/{pipe}/setup", mw.withAuth(handleRequest(c.PutPipeSetup))).Methods("PUT")
	v1.HandleFunc("/integrations/{service}/pipes/{pipe}/setup", mw.withAuth(handleRequest(c.PostPipeSetup))).Methods("POST")
	v1.HandleFunc("/integrations/{service}/pipes/{pipe}/setup", mw.withAuth(handleRequest(c.DeletePipeSetup))).Methods("DELETE")
	v1.HandleFunc("/integrations/{service}/pipes/{pipe}/log", mw.withService(mw.withAuth(handleRequest(c.GetServicePipeLog)))).Methods("GET")
	v1.HandleFunc("/integrations/{service}/pipes/{pipe}/clear_connections", mw.withService(mw.withAuth(handleRequest(c.PostServicePipeClearConnections)))).Methods("POST")

	v1.HandleFunc("/integrations/{service}/accounts", mw.withAuth(handleRequest(c.GetServiceAccounts))).Methods("GET")
	v1.HandleFunc("/integrations/{service}/auth_url", mw.withAuth(handleRequest(c.GetAuthURL))).Methods("GET")
	v1.HandleFunc("/integrations/{service}/authorizations", mw.withAuth(handleRequest(c.PostAuthorization))).Methods("POST")
	v1.HandleFunc("/integrations/{service}/authorizations", mw.withAuth(handleRequest(c.DeleteAuthorization))).Methods("DELETE")

	v1.HandleFunc("/integrations/{service}/pipes/{pipe}/users", mw.withAuth(handleRequest(c.GetServiceUsers))).Methods("GET")
	v1.HandleFunc("/integrations/{service}/pipes/{pipe}/run", mw.withService(mw.withAuth(handleRequest(c.PostPipeRun)))).Methods("POST")

	return router
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
	log.Fatal(http.ListenAndServe(listenAddress, http.DefaultServeMux))
}
