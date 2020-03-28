package server

import (
	"net/http"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"

	"github.com/toggl/pipes-api/pkg/domain"
	"github.com/toggl/pipes-api/pkg/integration"
)

type Middleware struct {
	domain.IntegrationsStorage
	domain.TogglClient
}

func NewMiddleware(clt domain.TogglClient, istore domain.IntegrationsStorage) *Middleware {
	return &Middleware{
		IntegrationsStorage: istore,
		TogglClient:         clt,
	}
}

func (mw *Middleware) withService(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serviceID := integration.ID(mux.Vars(r)["service"])
		if !mw.IntegrationsStorage.IsValidService(serviceID) {
			http.Error(w, "Missing or invalid service", http.StatusBadRequest)
			return
		}
		pipeID := integration.PipeID(mux.Vars(r)["pipe"])
		if !mw.IntegrationsStorage.IsValidPipe(pipeID) {
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
		workspaceID, err = mw.TogglClient.GetWorkspaceIdByToken(authData.Username)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		context.Set(r, workspaceIDKey, workspaceID)
		context.Set(r, workspaceTokenKey, authData.Username)
		handler(w, r)
	}
}
