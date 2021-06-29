package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/tambet/oauthplain"
)

// mutex to prevent multiple of postPipeRun on same workspace run at same time
var postPipeRunWorkspaceLock = map[int]*sync.Mutex{}
var postPipeRunLock sync.Mutex

type Selector struct {
	IDs         []int `json:"ids"`
	SendInvites bool  `json:"send_invites"`
}

func getIntegrations(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	integrations, err := workspaceIntegrations(workspaceID)
	if err != nil {
		return internalServerError(err.Error())
	}
	return ok(integrations)
}

func getIntegrationPipe(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID := mux.Vars(req.r)["service"]
	if !serviceType.MatchString(serviceID) {
		return badRequest("Missing or invalid service")
	}
	pipeID := mux.Vars(req.r)["pipe"]
	if !pipeType.MatchString(pipeID) {
		return badRequest("Missing or invalid pipe")
	}

	pipe, err := loadPipe(workspaceID, serviceID, pipeID)
	if err != nil {
		return internalServerError(err.Error())
	}
	if pipe == nil {
		pipe = NewPipe(workspaceID, serviceID, pipeID)
	}

	pipe.PipeStatus, err = loadPipeStatus(workspaceID, serviceID, pipeID)
	if err != nil {
		return internalServerError(err.Error())
	}

	return ok(pipe)
}

func postPipeSetup(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID := mux.Vars(req.r)["service"]
	if !serviceType.MatchString(serviceID) {
		return badRequest("Missing or invalid service")
	}
	pipeID := mux.Vars(req.r)["pipe"]
	if !pipeType.MatchString(pipeID) {
		return badRequest("Missing or invalid pipe")
	}

	pipe := NewPipe(workspaceID, serviceID, pipeID)
	errorMsg := pipe.validateServiceConfig(req.body)
	if errorMsg != "" {
		return badRequest(errorMsg)
	}

	if err := pipe.save(); err != nil {
		return internalServerError(err.Error())
	}
	return ok(nil)
}

func putPipeSetup(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID := mux.Vars(req.r)["service"]
	if !serviceType.MatchString(serviceID) {
		return badRequest("Missing or invalid service")
	}
	pipeID := mux.Vars(req.r)["pipe"]
	if !pipeType.MatchString(pipeID) {
		return badRequest("Missing or invalid pipe")
	}
	if len(req.body) == 0 {
		return badRequest("Missing payload")
	}
	pipe, err := loadPipe(workspaceID, serviceID, pipeID)
	if err != nil {
		return internalServerError(err.Error())
	}
	if pipe == nil {
		return badRequest("Pipe is not configured")
	}
	if err := json.Unmarshal(req.body, &pipe); err != nil {
		return internalServerError(err.Error())
	}
	if err := pipe.save(); err != nil {
		return internalServerError(err.Error())
	}
	return ok(nil)
}

func deletePipeSetup(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID := mux.Vars(req.r)["service"]
	if !serviceType.MatchString(serviceID) {
		return badRequest("Missing or invalid service")
	}
	pipeID := mux.Vars(req.r)["pipe"]
	if !pipeType.MatchString(pipeID) {
		return badRequest("Missing or invalid pipe")
	}
	pipe, err := loadPipe(workspaceID, serviceID, pipeID)
	if err != nil {
		return internalServerError(err.Error())
	}
	if pipe == nil {
		return badRequest("Pipe is not configured")
	}
	if err := pipe.destroy(workspaceID); err != nil {
		return internalServerError(err.Error())
	}
	return ok(nil)
}

func getAuthURL(req Request) Response {
	serviceID := mux.Vars(req.r)["service"]
	accountName := req.r.FormValue("account_name")
	callbackURL := req.r.FormValue("callback_url")

	if !serviceType.MatchString(serviceID) {
		return badRequest("Missing or invalid service")
	}
	if accountName == "" {
		return badRequest("Missing or invalid account_name")
	}
	if callbackURL == "" {
		return badRequest("Missing or invalid callback_url")
	}

	config, found := oAuth1Configs[serviceID]
	if !found {
		return badRequest("Service OAuth config not found")
	}
	transport := &oauthplain.Transport{
		Config: config.UpdateURLs(accountName),
	}
	token, err := transport.AuthCodeURL(callbackURL)
	if err != nil {
		return internalServerError(err.Error())
	}
	return ok(struct {
		AuthURL string `json:"auth_url"`
	}{
		token.AuthorizeUrl,
	})
}

func postAuthorization(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID := mux.Vars(req.r)["service"]
	if !serviceType.MatchString(serviceID) {
		return badRequest("Missing or invalid service")
	}
	if len(req.body) == 0 {
		return badRequest("Missing payload")
	}

	var payload map[string]interface{}
	err := json.Unmarshal(req.body, &payload)
	if err != nil {
		return internalServerError(err.Error())
	}

	authorization := NewAuthorization(workspaceID, serviceID)
	authorization.WorkspaceToken = currentWorkspaceToken(req.r)

	for _, integration := range availableIntegrations {
		if serviceID == integration.ID && integration.Deprecated {
			return badRequest(fmt.Sprintf("Deprecated %s integration is not available for new authorizations", serviceID))
		}
	}

	switch availableAuthorizations[serviceID] {
	case "oauth1":
		authorization.Data, err = oAuth1Exchange(serviceID, payload)
	case "oauth2":
		authorization.Data, err = oAuth2Exchange(serviceID, payload)
	}
	if err != nil {
		return internalServerError(err.Error())
	}

	if err := authorization.save(); err != nil {
		return internalServerError(err.Error())
	}
	return ok(nil)
}

func deleteAuthorization(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID := mux.Vars(req.r)["service"]
	if !serviceType.MatchString(serviceID) {
		return badRequest("Missing or invalid service")
	}
	service := getService(serviceID, workspaceID)
	authorization, err := loadAuth(service)
	if err != nil {
		return internalServerError(err.Error())
	}
	if err := authorization.destroy(service); err != nil {
		return internalServerError(err.Error())
	}
	_, err = db.Exec(deletePipeSQL, workspaceID, serviceID+"%")
	if err != nil {
		return internalServerError(err.Error())
	}
	return ok(nil)
}

func getServiceAccounts(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID := mux.Vars(req.r)["service"]
	if !serviceType.MatchString(serviceID) {
		return badRequest("Missing or invalid service")
	}
	service := getService(serviceID, workspaceID)
	auth, err := loadAuth(service)
	if err != nil {
		return badRequest("No authorizations for " + serviceID)
	}
	if err := auth.refresh(); err != nil {
		return badRequest("oAuth refresh failed!")
	}
	forceImport := req.r.FormValue("force")
	if forceImport == "true" {
		if err := clearImportFor(service, "accounts"); err != nil {
			return internalServerError(err.Error())
		}
	}
	accountsResponse, err := getAccounts(service)
	if err != nil {
		return internalServerError("Unable to get accounts from DB")
	}
	if accountsResponse == nil {
		go func() {
			if err := fetchAccounts(service); err != nil {
				log.Print(err.Error())
			}
		}()
		return noContent()
	}
	return ok(accountsResponse)
}

func getServiceUsers(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)

	serviceID := mux.Vars(req.r)["service"]
	if !serviceType.MatchString(serviceID) {
		return badRequest("Missing or invalid service")
	}
	service := getService(serviceID, workspaceID)
	if _, err := loadAuth(service); err != nil {
		return badRequest("No authorizations for " + serviceID)
	}
	pipeID := "users"
	pipe, err := loadPipe(workspaceID, serviceID, pipeID)
	if err != nil {
		return internalServerError(err.Error())
	}
	if pipe == nil {
		return badRequest("Pipe is not configured")
	}
	if err := service.setParams(pipe.ServiceParams); err != nil {
		return badRequest(err.Error())
	}

	forceImport := req.r.FormValue("force")
	if forceImport == "true" {
		if err := clearImportFor(service, pipeID); err != nil {
			return internalServerError(err.Error())
		}
	}

	usersResponse, err := getUsers(service)
	if err != nil {
		return internalServerError("Unable to get users from DB")
	}
	if usersResponse == nil {
		if forceImport == "true" {
			go func() {
				if err := pipe.fetchObjects(false); err != nil {
					log.Print(err.Error())
				}
			}()
		}
		return noContent()
	}
	return ok(usersResponse)
}

func getServicePipeLog(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID, pipeID := currentServicePipeID(req.r)

	pipeStatus, err := loadPipeStatus(workspaceID, serviceID, pipeID)
	if err != nil {
		return internalServerError("Unable to get log from DB")
	}
	if pipeStatus == nil {
		return noContent()
	}
	return Response{http.StatusOK, pipeStatus.generateLog(), "text/plain"}
}

func postServicePipeClearConnections(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID, pipeID := currentServicePipeID(req.r)

	pipe, err := loadPipe(workspaceID, serviceID, pipeID)
	if err != nil {
		return internalServerError(err.Error())
	}
	if pipe == nil {
		return badRequest("Pipe is not configured")
	}

	err = pipe.clearPipeConnections()
	if err != nil {
		return internalServerError("Unable to get clear connections")
	}
	return noContent()
}

func postPipeRun(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)

	// make sure no race condition on fetching workspace lock
	postPipeRunLock.Lock()
	workspaceLock, exists := postPipeRunWorkspaceLock[workspaceID]
	if !exists {
		workspaceLock = &sync.Mutex{}
		postPipeRunWorkspaceLock[workspaceID] = workspaceLock
	}
	postPipeRunLock.Unlock()

	serviceID, pipeID := currentServicePipeID(req.r)

	pipe, err := loadPipe(workspaceID, serviceID, pipeID)
	if err != nil {
		return internalServerError(err.Error())
	}
	if pipe == nil {
		return badRequest("Pipe is not configured")
	}
	if msg := pipe.validatePayload(req.body); msg != "" {
		return badRequest(msg)
	}
	if pipe.ID == "users" {
		go func() {
			workspaceLock.Lock()
			pipe.run()
			workspaceLock.Unlock()
		}()
		time.Sleep(500 * time.Millisecond)
	} else {
		_, err = db.Exec(queuePipeAsFirstSQL, pipe.workspaceID, pipe.key)
		if err != nil {
			return internalServerError(err.Error())
		}
	}
	return ok(nil)
}

func getStatus(req Request) Response {
	resp := &struct {
		Reasons []string `json:"reasons"`
	}{}

	if dbIsDown() {
		resp.Reasons = append(resp.Reasons, "Database is down")
	}

	if err := pingTogglAPI(); err != nil {
		resp.Reasons = append(resp.Reasons, err.Error())
	}

	if len(resp.Reasons) > 0 {
		return serviceUnavailable(resp)
	}
	return ok(map[string]string{"status": "OK"})
}
