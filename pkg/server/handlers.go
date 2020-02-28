package server

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/tambet/oauthplain"

	"github.com/toggl/pipes-api/pkg/environment"
	"github.com/toggl/pipes-api/pkg/storage"
)

// mutex to prevent multiple of postPipeRun on same workspace run at same time
var postPipeRunWorkspaceLock = map[int]*sync.Mutex{}
var postPipeRunLock sync.Mutex

type Controller struct {
	stResolver ServiceTypeResolver
	ptResolver PipeTypeResolver

	AuthorizationStorage *storage.AuthorizationStorage
	ConnectionStorage    *storage.ConnectionStorage
	PipesStorage         *storage.PipesStorage
	Environment          *environment.Environment
}

func NewController(env *environment.Environment, pipes *storage.PipesStorage) *Controller {
	return &Controller{
		AuthorizationStorage: pipes.AuthorizationStorage,
		ConnectionStorage:    pipes.ConnectionStorage,
		PipesStorage:         pipes,
		Environment:          env,

		stResolver: pipes,
		ptResolver: pipes,
	}
}

func (c *Controller) GetIntegrations(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	resp, err := c.PipesStorage.WorkspaceIntegrations(workspaceID)
	if err != nil {
		return internalServerError(err.Error())
	}
	return ok(resp)
}

func (c *Controller) GetIntegrationPipe(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID := mux.Vars(req.r)["service"]
	if !c.stResolver.AvailableServiceType(serviceID) {
		return badRequest("Missing or invalid service")
	}
	pipeID := mux.Vars(req.r)["pipe"]
	if !c.ptResolver.AvailablePipeType(pipeID) {
		return badRequest("Missing or invalid pipe")
	}

	pipe, err := c.PipesStorage.LoadPipe(workspaceID, serviceID, pipeID)
	if err != nil {
		return internalServerError(err.Error())
	}
	if pipe == nil {
		pipe = environment.NewPipe(workspaceID, serviceID, pipeID)
	}

	pipe.PipeStatus, err = c.PipesStorage.LoadPipeStatus(workspaceID, serviceID, pipeID)
	if err != nil {
		return internalServerError(err.Error())
	}

	return ok(pipe)
}

func (c *Controller) PostPipeSetup(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID := mux.Vars(req.r)["service"]
	if !c.stResolver.AvailableServiceType(serviceID) {
		return badRequest("Missing or invalid service")
	}
	pipeID := mux.Vars(req.r)["pipe"]
	if !c.ptResolver.AvailablePipeType(pipeID) {
		return badRequest("Missing or invalid pipe")
	}

	pipe := environment.NewPipe(workspaceID, serviceID, pipeID)
	errorMsg := pipe.ValidateServiceConfig(req.body)
	if errorMsg != "" {
		return badRequest(errorMsg)
	}

	if err := c.PipesStorage.Save(pipe); err != nil {
		return internalServerError(err.Error())
	}
	return ok(nil)
}

func (c *Controller) PutPipeSetup(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID := mux.Vars(req.r)["service"]
	if !c.stResolver.AvailableServiceType(serviceID) {
		return badRequest("Missing or invalid service")
	}
	pipeID := mux.Vars(req.r)["pipe"]
	if !c.ptResolver.AvailablePipeType(pipeID) {
		return badRequest("Missing or invalid pipe")
	}
	if len(req.body) == 0 {
		return badRequest("Missing payload")
	}
	pipe, err := c.PipesStorage.LoadPipe(workspaceID, serviceID, pipeID)
	if err != nil {
		return internalServerError(err.Error())
	}
	if pipe == nil {
		return badRequest("PipeConfig is not configured")
	}
	if err := json.Unmarshal(req.body, &pipe); err != nil {
		return internalServerError(err.Error())
	}
	if err := c.PipesStorage.Save(pipe); err != nil {
		return internalServerError(err.Error())
	}
	return ok(nil)
}

func (c *Controller) DeletePipeSetup(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID := mux.Vars(req.r)["service"]
	if !c.stResolver.AvailableServiceType(serviceID) {
		return badRequest("Missing or invalid service")
	}
	pipeID := mux.Vars(req.r)["pipe"]
	if !c.ptResolver.AvailablePipeType(pipeID) {
		return badRequest("Missing or invalid pipe")
	}
	pipe, err := c.PipesStorage.LoadPipe(workspaceID, serviceID, pipeID)
	if err != nil {
		return internalServerError(err.Error())
	}
	if pipe == nil {
		return badRequest("PipeConfig is not configured")
	}
	if err := c.PipesStorage.Destroy(pipe, workspaceID); err != nil {
		return internalServerError(err.Error())
	}
	return ok(nil)
}

func (c *Controller) GetAuthURL(req Request) Response {
	serviceID := mux.Vars(req.r)["service"]
	accountName := req.r.FormValue("account_name")
	callbackURL := req.r.FormValue("callback_url")

	if !c.stResolver.AvailableServiceType(serviceID) {
		return badRequest("Missing or invalid service")
	}
	if accountName == "" {
		return badRequest("Missing or invalid account_name")
	}
	if callbackURL == "" {
		return badRequest("Missing or invalid callback_url")
	}
	config, found := c.Environment.GetOAuth1Configs(serviceID)
	if !found {
		return badRequest("Environment OAuth config not found")
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
	if !c.stResolver.AvailableServiceType(serviceID) {
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

	authorization := environment.NewAuthorization(workspaceID, serviceID)
	authorization.WorkspaceToken = currentWorkspaceToken(req.r)

	switch c.Environment.GetAvailableAuthorizations(serviceID) {
	case "oauth1":
		authorization.Data, err = c.Environment.OAuth1Exchange(serviceID, payload)
	case "oauth2":
		authorization.Data, err = c.Environment.OAuth2Exchange(serviceID, payload)
	}
	if err != nil {
		return internalServerError(err.Error())
	}

	if err := c.AuthorizationStorage.Save(authorization); err != nil {
		return internalServerError(err.Error())
	}
	return ok(nil)
}

func (c *Controller) DeleteAuthorization(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID := mux.Vars(req.r)["service"]
	if !c.stResolver.AvailableServiceType(serviceID) {
		return badRequest("Missing or invalid service")
	}
	service := environment.Create(serviceID, workspaceID)
	_, err := c.AuthorizationStorage.LoadAuth(service)
	if err != nil {
		return internalServerError(err.Error())
	}
	if err := c.AuthorizationStorage.Destroy(service); err != nil {
		return internalServerError(err.Error())
	}
	if err := c.PipesStorage.DeletePipeByWorkspaceIDServiceID(workspaceID, serviceID); err != nil {
		return internalServerError(err.Error())
	}
	return ok(nil)
}

func (c *Controller) GetServiceAccounts(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID := mux.Vars(req.r)["service"]
	if !c.stResolver.AvailableServiceType(serviceID) {
		return badRequest("Missing or invalid service")
	}
	service := environment.Create(serviceID, workspaceID)
	auth, err := c.AuthorizationStorage.LoadAuth(service)
	if err != nil {
		return badRequest("No authorizations for " + serviceID)
	}
	if err := c.AuthorizationStorage.Refresh(auth); err != nil {
		return badRequest("oAuth refresh failed!")
	}
	forceImport := req.r.FormValue("force")
	if forceImport == "true" {
		if err := c.PipesStorage.ClearImportFor(service, "accounts"); err != nil {
			return internalServerError(err.Error())
		}
	}
	accountsResponse, err := c.PipesStorage.GetAccounts(service)
	if err != nil {
		return internalServerError("Unable to get accounts from DB")
	}
	if accountsResponse == nil {
		go func() {
			if err := c.PipesStorage.FetchAccounts(service); err != nil {
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
	if !c.stResolver.AvailableServiceType(serviceID) {
		return badRequest("Missing or invalid service")
	}
	service := environment.Create(serviceID, workspaceID)
	if _, err := c.AuthorizationStorage.LoadAuth(service); err != nil {
		return badRequest("No authorizations for " + serviceID)
	}
	pipeID := "users"
	pipe, err := c.PipesStorage.LoadPipe(workspaceID, serviceID, pipeID)
	if err != nil {
		return internalServerError(err.Error())
	}
	if pipe == nil {
		return badRequest("PipeConfig is not configured")
	}
	if err := service.SetParams(pipe.ServiceParams); err != nil {
		return badRequest(err.Error())
	}

	forceImport := req.r.FormValue("force")
	if forceImport == "true" {
		if err := c.PipesStorage.ClearImportFor(service, pipeID); err != nil {
			return internalServerError(err.Error())
		}
	}

	usersResponse, err := c.PipesStorage.GetUsers(service)
	if err != nil {
		return internalServerError("Unable to get users from DB")
	}
	if usersResponse == nil {
		if forceImport == "true" {
			go func() {
				if err := c.PipesStorage.FetchObjects(pipe, false); err != nil {
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

	pipeStatus, err := c.PipesStorage.LoadPipeStatus(workspaceID, serviceID, pipeID)
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

	pipe, err := c.PipesStorage.LoadPipe(workspaceID, serviceID, pipeID)
	if err != nil {
		return internalServerError(err.Error())
	}
	if pipe == nil {
		return badRequest("PipeConfig is not configured")
	}

	err = c.PipesStorage.ClearPipeConnections(pipe)
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

	pipe, err := c.PipesStorage.LoadPipe(workspaceID, serviceID, pipeID)
	if err != nil {
		return internalServerError(err.Error())
	}
	if pipe == nil {
		return badRequest("PipeConfig is not configured")
	}
	if msg := pipe.ValidatePayload(req.body); msg != "" {
		return badRequest(msg)
	}
	if pipe.ID == "users" {
		go func() {
			workspaceLock.Lock()
			c.PipesStorage.Run(pipe)
			workspaceLock.Unlock()
		}()
		time.Sleep(500 * time.Millisecond) // TODO: Is that synchronization ? :D
	} else {
		if err := c.PipesStorage.QueuePipeAsFirst(pipe); err != nil {
			return internalServerError(err.Error())
		}
	}
	return ok(nil)
}

func (c *Controller) GetStatus(req Request) Response {
	resp := &struct {
		Reasons []string `json:"reasons"`
	}{}

	if c.PipesStorage.IsDown() {
		resp := &struct {
			Reasons []string `json:"reasons"`
		}{
			[]string{"Database is down"},
		}
		return serviceUnavailable(resp)
	}

	if err := pingTogglAPI(); err != nil {
		resp.Reasons = append(resp.Reasons, err.Error())
	}

	if len(resp.Reasons) > 0 {
		return serviceUnavailable(resp)
	}
	return ok(map[string]string{"status": "OK"})
}
