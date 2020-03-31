package server

import (
	"log"
	"net/http"

	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	gouuid "github.com/nu7hatch/gouuid"

	"github.com/toggl/pipes-api/pkg/domain"
)

type Middleware struct {
	integrationsStorage domain.IntegrationsStorage
	togglClient         domain.TogglClient
}

func NewMiddleware(integrationsStorage domain.IntegrationsStorage, togglClient domain.TogglClient) *Middleware {
	if integrationsStorage == nil {
		panic("Middleware.integrationsStorage should not be nil")
	}
	if togglClient == nil {
		panic("Middleware.togglClient should not be nil")
	}
	return &Middleware{
		integrationsStorage: integrationsStorage,
		togglClient:         togglClient,
	}
}

func (mw *Middleware) withService(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		serviceID := domain.IntegrationID(mux.Vars(r)["service"])
		if !mw.integrationsStorage.IsValidService(serviceID) {
			http.Error(w, "Missing or invalid service", http.StatusBadRequest)
			return
		}
		pipeID := domain.PipeID(mux.Vars(r)["pipe"])
		if !mw.integrationsStorage.IsValidPipe(pipeID) {
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
		workspaceID, err = mw.togglClient.GetWorkspaceIdByToken(authData.Username)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		context.Set(r, workspaceIDKey, workspaceID)
		context.Set(r, workspaceTokenKey, authData.Username)
		handler(w, r)
	}
}

func (mw *Middleware) withUUID(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u4, err := gouuid.NewV4()
		if err != nil {
			log.Print(err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		uuidToken := u4.String()
		context.Set(r, uuidKey, uuidToken)
		handler(w, r)
	}
}
