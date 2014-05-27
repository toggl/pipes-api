package main

import (
	"code.google.com/p/goauth2/oauth"
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http"
	"regexp"
)

type Selector struct {
	IDs []int `json:"ids"`
}

var serviceType = regexp.MustCompile("basecamp|freshbooks")
var pipeType = regexp.MustCompile("users|projects|todolists|todos")

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
	if err := json.Unmarshal(req.body, &pipe); err != nil {
		return internalServerError(err.Error())
	}

	if errorMsg := pipe.validate(); errorMsg != "" {
		return badRequest(errorMsg)
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
	return ok(nil)
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

	var response map[string]interface{}
	if err := json.Unmarshal(req.body, &response); err != nil {
		return internalServerError(err.Error())
	}

	code := response["code"].(string)
	if code == "" {
		return badRequest("Missing code")
	}

	config, res := knownOauthConfigs[serviceID+"_"+*environment]
	if !res {
		return badRequest("Oauth config not found!")
	}
	transport := &oauth.Transport{Config: config}
	token, err := transport.Exchange(code)
	if err != nil {
		return badRequest(err.Error())
	}

	auth := Authorization{
		AccessToken:    token.AccessToken,
		RefreshToken:   token.RefreshToken,
		Expiry:         token.Expiry,
		WorkspaceToken: currentWorkspaceToken(req.r),
	}

	if err := auth.save(workspaceID, serviceID); err != nil {
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
	return ok(nil)
}

func getServiceAccounts(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID := mux.Vars(req.r)["service"]
	if !serviceType.MatchString(serviceID) {
		return badRequest("Missing or invalid service")
	}
	service := getService(serviceID, workspaceID)

	if serviceNotAuthorized(service) {
		return badRequest("No authorizations for " + serviceID)
	}
	accountsResponse, err := getAccounts(service)
	if err != nil {
		return internalServerError("Unable to get accounts from DB")
	}
	if accountsResponse == nil {
		go fetchAccounts(service)
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
	if serviceNotAuthorized(service) {
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
	service.setAccount(pipe.AccountID)

	forceImport := req.r.FormValue("force")
	if forceImport == "true" {
		clearImportFor(service, pipeID)
	}

	usersResponse, err := getUsers(service)
	if err != nil {
		return internalServerError("Unable to get users from DB")
	}
	if usersResponse == nil {
		if forceImport == "true" {
			go pipe.fetchObjects(false)
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

func postPipeRun(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
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

	go pipe.run()
	return ok(nil)
}

type (
	statusResponse struct {
		Reasons []string `json:"reasons"`
	}
)

func getStatus(req Request) Response {
	if dbIsDown() {
		resp := &statusResponse{}
		resp.Reasons = make([]string, 1)
		resp.Reasons[0] = "Database is down"
		return serviceUnavailable(resp)
	}
	return ok("OK")
}
