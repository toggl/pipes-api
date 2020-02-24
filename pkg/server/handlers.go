package server

import (
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/tambet/oauthplain"

	"github.com/toggl/pipes-api/pkg/cfg"
	"github.com/toggl/pipes-api/pkg/integrations"
	"github.com/toggl/pipes-api/pkg/pipes"
	"github.com/toggl/pipes-api/pkg/storage"
)

// mutex to prevent multiple of postPipeRun on same workspace run at same time
var postPipeRunWorkspaceLock = map[int]*sync.Mutex{}
var postPipeRunLock sync.Mutex

type Controller struct {
	Storage              *storage.Storage
	AuthorizationService *pipes.AuthorizationService
	ConnectionService    *pipes.ConnectionService
	OAuthService         *cfg.OAuthService
	PipeService          *pipes.PipeService

	AvailablePipeType    *regexp.Regexp
	AvailableServiceType *regexp.Regexp
}

func (c *Controller) GetIntegrations(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	resp, err := c.PipeService.WorkspaceIntegrations(workspaceID)
	if err != nil {
		return internalServerError(err.Error())
	}
	return ok(resp)
}

func (c *Controller) GetIntegrationPipe(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID := mux.Vars(req.r)["service"]
	if !c.AvailableServiceType.MatchString(serviceID) {
		return badRequest("Missing or invalid service")
	}
	pipeID := mux.Vars(req.r)["pipe"]
	if !c.AvailablePipeType.MatchString(pipeID) {
		return badRequest("Missing or invalid pipe")
	}

	pipe, err := c.PipeService.LoadPipe(workspaceID, serviceID, pipeID)
	if err != nil {
		return internalServerError(err.Error())
	}
	if pipe == nil {
		pipe = cfg.NewPipe(workspaceID, serviceID, pipeID)
	}

	pipe.PipeStatus, err = c.PipeService.LoadPipeStatus(workspaceID, serviceID, pipeID)
	if err != nil {
		return internalServerError(err.Error())
	}

	return ok(pipe)
}

func (c *Controller) PostPipeSetup(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID := mux.Vars(req.r)["service"]
	if !c.AvailableServiceType.MatchString(serviceID) {
		return badRequest("Missing or invalid service")
	}
	pipeID := mux.Vars(req.r)["pipe"]
	if !c.AvailablePipeType.MatchString(pipeID) {
		return badRequest("Missing or invalid pipe")
	}

	pipe := cfg.NewPipe(workspaceID, serviceID, pipeID)
	errorMsg := pipe.ValidateServiceConfig(req.body)
	if errorMsg != "" {
		return badRequest(errorMsg)
	}

	if err := c.PipeService.Save(pipe); err != nil {
		return internalServerError(err.Error())
	}
	return ok(nil)
}

func (c *Controller) PutPipeSetup(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID := mux.Vars(req.r)["service"]
	if !c.AvailableServiceType.MatchString(serviceID) {
		return badRequest("Missing or invalid service")
	}
	pipeID := mux.Vars(req.r)["pipe"]
	if !c.AvailablePipeType.MatchString(pipeID) {
		return badRequest("Missing or invalid pipe")
	}
	if len(req.body) == 0 {
		return badRequest("Missing payload")
	}
	pipe, err := c.PipeService.LoadPipe(workspaceID, serviceID, pipeID)
	if err != nil {
		return internalServerError(err.Error())
	}
	if pipe == nil {
		return badRequest("Pipe is not configured")
	}
	if err := json.Unmarshal(req.body, &pipe); err != nil {
		return internalServerError(err.Error())
	}
	if err := c.PipeService.Save(pipe); err != nil {
		return internalServerError(err.Error())
	}
	return ok(nil)
}

func (c *Controller) DeletePipeSetup(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID := mux.Vars(req.r)["service"]
	if !c.AvailableServiceType.MatchString(serviceID) {
		return badRequest("Missing or invalid service")
	}
	pipeID := mux.Vars(req.r)["pipe"]
	if !c.AvailablePipeType.MatchString(pipeID) {
		return badRequest("Missing or invalid pipe")
	}
	pipe, err := c.PipeService.LoadPipe(workspaceID, serviceID, pipeID)
	if err != nil {
		return internalServerError(err.Error())
	}
	if pipe == nil {
		return badRequest("Pipe is not configured")
	}
	if err := c.PipeService.Destroy(pipe, workspaceID); err != nil {
		return internalServerError(err.Error())
	}
	return ok(nil)
}

func (c *Controller) GetAuthURL(req Request) Response {
	serviceID := mux.Vars(req.r)["service"]
	accountName := req.r.FormValue("account_name")
	callbackURL := req.r.FormValue("callback_url")

	if !c.AvailableServiceType.MatchString(serviceID) {
		return badRequest("Missing or invalid service")
	}
	if accountName == "" {
		return badRequest("Missing or invalid account_name")
	}
	if callbackURL == "" {
		return badRequest("Missing or invalid callback_url")
	}
	config, found := c.OAuthService.GetOAuth1Configs(serviceID)
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

func (c *Controller) PostAuthorization(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID := mux.Vars(req.r)["service"]
	if !c.AvailableServiceType.MatchString(serviceID) {
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

	authorization := cfg.NewAuthorization(workspaceID, serviceID)
	authorization.WorkspaceToken = currentWorkspaceToken(req.r)

	switch c.OAuthService.GetAvailableAuthorizations(serviceID) {
	case "oauth1":
		authorization.Data, err = c.OAuthService.OAuth1Exchange(serviceID, payload)
	case "oauth2":
		authorization.Data, err = c.OAuthService.OAuth2Exchange(serviceID, payload)
	}
	if err != nil {
		return internalServerError(err.Error())
	}

	if err := c.AuthorizationService.Save(authorization); err != nil {
		return internalServerError(err.Error())
	}
	return ok(nil)
}

func (c *Controller) DeleteAuthorization(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID := mux.Vars(req.r)["service"]
	if !c.AvailableServiceType.MatchString(serviceID) {
		return badRequest("Missing or invalid service")
	}
	service := integrations.GetService(serviceID, workspaceID)
	_, err := c.AuthorizationService.LoadAuth(service)
	if err != nil {
		return internalServerError(err.Error())
	}
	if err := c.AuthorizationService.Destroy(service); err != nil {
		return internalServerError(err.Error())
	}
	if err := c.PipeService.DeletePipeByWorkspaceIDServiceID(workspaceID, serviceID); err != nil {
		return internalServerError(err.Error())
	}
	return ok(nil)
}

func (c *Controller) GetServiceAccounts(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID := mux.Vars(req.r)["service"]
	if !c.AvailableServiceType.MatchString(serviceID) {
		return badRequest("Missing or invalid service")
	}
	service := integrations.GetService(serviceID, workspaceID)
	auth, err := c.AuthorizationService.LoadAuth(service)
	if err != nil {
		return badRequest("No authorizations for " + serviceID)
	}
	if err := c.AuthorizationService.Refresh(auth); err != nil {
		return badRequest("oAuth refresh failed!")
	}
	forceImport := req.r.FormValue("force")
	if forceImport == "true" {
		if err := c.PipeService.ClearImportFor(service, "accounts"); err != nil {
			return internalServerError(err.Error())
		}
	}
	accountsResponse, err := c.PipeService.GetAccounts(service)
	if err != nil {
		return internalServerError("Unable to get accounts from DB")
	}
	if accountsResponse == nil {
		go func() {
			if err := c.PipeService.FetchAccounts(service); err != nil {
				log.Print(err.Error())
			}
		}()
		return noContent()
	}
	return ok(accountsResponse)
}

func (c *Controller) GetServiceUsers(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)

	serviceID := mux.Vars(req.r)["service"]
	if !c.AvailableServiceType.MatchString(serviceID) {
		return badRequest("Missing or invalid service")
	}
	service := integrations.GetService(serviceID, workspaceID)
	if _, err := c.AuthorizationService.LoadAuth(service); err != nil {
		return badRequest("No authorizations for " + serviceID)
	}
	pipeID := "users"
	pipe, err := c.PipeService.LoadPipe(workspaceID, serviceID, pipeID)
	if err != nil {
		return internalServerError(err.Error())
	}
	if pipe == nil {
		return badRequest("Pipe is not configured")
	}
	if err := service.SetParams(pipe.ServiceParams); err != nil {
		return badRequest(err.Error())
	}

	forceImport := req.r.FormValue("force")
	if forceImport == "true" {
		if err := c.PipeService.ClearImportFor(service, pipeID); err != nil {
			return internalServerError(err.Error())
		}
	}

	usersResponse, err := c.PipeService.GetUsers(service)
	if err != nil {
		return internalServerError("Unable to get users from DB")
	}
	if usersResponse == nil {
		if forceImport == "true" {
			go func() {
				if err := c.PipeService.FetchObjects(pipe, false); err != nil {
					log.Print(err.Error())
				}
			}()
		}
		return noContent()
	}
	return ok(usersResponse)
}

func (c *Controller) GetServicePipeLog(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID, pipeID := currentServicePipeID(req.r)

	pipeStatus, err := c.PipeService.LoadPipeStatus(workspaceID, serviceID, pipeID)
	if err != nil {
		return internalServerError("Unable to get log from DB")
	}
	if pipeStatus == nil {
		return noContent()
	}
	return Response{http.StatusOK, pipeStatus.GenerateLog(), "text/plain"}
}

func (c *Controller) PostServicePipeClearConnections(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID, pipeID := currentServicePipeID(req.r)

	pipe, err := c.PipeService.LoadPipe(workspaceID, serviceID, pipeID)
	if err != nil {
		return internalServerError(err.Error())
	}
	if pipe == nil {
		return badRequest("Pipe is not configured")
	}

	err = c.PipeService.ClearPipeConnections(pipe)
	if err != nil {
		return internalServerError("Unable to get clear connections")
	}
	return noContent()
}

func (c *Controller) PostPipeRun(req Request) Response {
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

	pipe, err := c.PipeService.LoadPipe(workspaceID, serviceID, pipeID)
	if err != nil {
		return internalServerError(err.Error())
	}
	if pipe == nil {
		return badRequest("Pipe is not configured")
	}
	if msg := pipe.ValidatePayload(req.body); msg != "" {
		return badRequest(msg)
	}
	if pipe.ID == "users" {
		go func() {
			workspaceLock.Lock()
			c.PipeService.Run(pipe)
			workspaceLock.Unlock()
		}()
		time.Sleep(500 * time.Millisecond) // TODO: Is that syncronization ??? Should be refactored!
	} else {
		if err := c.PipeService.QueuePipeAsFirst(pipe); err != nil {
			return internalServerError(err.Error())
		}
	}
	return ok(nil)
}

func (c *Controller) GetStatus(req Request) Response {
	if c.Storage.IsDown() {
		resp := &struct {
			Reasons []string `json:"reasons"`
		}{
			[]string{"Database is down"},
		}
		return serviceUnavailable(resp)
	}
	return ok("OK")
}
