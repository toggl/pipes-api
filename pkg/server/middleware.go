package server

import (
	"net/http"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"

	"github.com/toggl/pipes-api/pkg/toggl"
)

type Middleware struct {
	stResolver ServiceTypeResolver
	ptResolver PipeTypeResolver
	api        *toggl.ApiClient
}

func NewMiddleware(api *toggl.ApiClient, str ServiceTypeResolver, ptr PipeTypeResolver) *Middleware {
	return &Middleware{
		stResolver: str,
		ptResolver: ptr,
		api:        api,
	}
}

func (mw *Middleware) withService(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serviceID := mux.Vars(r)["service"]
		if !mw.stResolver.AvailableServiceType(serviceID) {
			http.Error(w, "Missing or invalid service", http.StatusBadRequest)
			return
		}
		pipeID := mux.Vars(r)["pipe"]
		if !mw.ptResolver.AvailablePipeType(pipeID) {
			http.Error(w, "Missing or invalid pipe", http.StatusBadRequest)
			return
		}
		context.Set(r, serviceIDKey, serviceID)
		context.Set(r, pipeIDKey, pipeID)
		handler(w, r)
	}
}

func (mw *Middleware) withAuth(handler http.HandlerFunc) http.HandlerFunc {
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
		workspaceID, err = mw.api.WithAuthToken(authData.Username).GetWorkspaceID()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		context.Set(r, workspaceIDKey, workspaceID)
		context.Set(r, workspaceTokenKey, authData.Username)
		handler(w, r)
	}
}