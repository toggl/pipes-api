package server

import (
	"net/http"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"

	"github.com/toggl/pipes-api/pkg/integrations"
	"github.com/toggl/pipes-api/pkg/pipe"
)

type Middleware struct {
	svc pipe.Service
	clt pipe.TogglClient
}

func NewMiddleware(clt pipe.TogglClient, svc pipe.Service) *Middleware {
	return &Middleware{
		svc: svc,
		clt: clt,
	}
}

func (mw *Middleware) withService(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serviceID := integrations.ExternalServiceID(mux.Vars(r)["service"])
		if !mw.svc.AvailableServiceType(serviceID) {
			http.Error(w, "Missing or invalid service", http.StatusBadRequest)
			return
		}
		pipeID := integrations.PipeID(mux.Vars(r)["pipe"])
		if !mw.svc.AvailablePipeType(pipeID) {
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
		workspaceID, err = mw.clt.GetWorkspaceIdByToken(authData.Username)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		context.Set(r, workspaceIDKey, workspaceID)
		context.Set(r, workspaceTokenKey, authData.Username)
		handler(w, r)
	}
}
