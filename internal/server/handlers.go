package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/toggl/pipes-api/internal/service"
	"github.com/toggl/pipes-api/pkg/domain"
	"github.com/toggl/pipes-api/pkg/integration"
)

type Controller struct {
	domain.PipeService
	domain.IntegrationsStorage
	domain.Queue
	Params
}

type Params struct {
	Version   string
	Revision  string
	BuildTime string
}

func (c *Controller) GetIntegrationsHandler(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	resp, err := c.PipeService.GetIntegrations(workspaceID)
	if err != nil {
		return internalServerError(err.Error())
	}
	return ok(resp)
}

func (c *Controller) ReadPipeHandler(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID, pipeID, err := c.getIntegrationParams(req)
	if err != nil {
		return badRequest(err.Error())
	}
	p, err := c.PipeService.GetPipe(workspaceID, serviceID, pipeID)
	if err != nil {
		return internalServerError(err.Error())
	}
	return ok(p)
}

func (c *Controller) CreatePipeHandler(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID, pipeID, err := c.getIntegrationParams(req)
	if err != nil {
		return badRequest(err.Error())
	}
	err = c.PipeService.CreatePipe(workspaceID, serviceID, pipeID, req.body)
	if err != nil {
		if errors.As(err, &service.SetParamsError{}) {
			return badRequest(err.Error())
		}
		return internalServerError(err.Error())
	}
	return ok(nil)
}

func (c *Controller) UpdatePipeHandler(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID, pipeID, err := c.getIntegrationParams(req)
	if err != nil {
		return badRequest(err.Error())
	}
	if len(req.body) == 0 {
		return badRequest("Missing payload")
	}

	err = c.PipeService.UpdatePipe(workspaceID, serviceID, pipeID, req.body)
	if err != nil {
		if errors.Is(err, service.ErrPipeNotConfigured) {
			return badRequest(err.Error())
		}
		return internalServerError(err.Error())
	}
	return ok(nil)
}

func (c *Controller) DeletePipeHandler(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID, pipeID, err := c.getIntegrationParams(req)
	if err != nil {
		return badRequest(err.Error())
	}
	err = c.PipeService.DeletePipe(workspaceID, serviceID, pipeID)
	if err != nil {
		if errors.Is(err, service.ErrPipeNotConfigured) {
			return badRequest(err.Error())
		}
		return internalServerError(err.Error())
	}
	return ok(nil)
}

func (c *Controller) GetAuthURLHandler(req Request) Response {
	serviceID, err := c.getServiceId(req)
	if err != nil {
		return badRequest(err.Error())
	}
	accountName := req.r.FormValue("account_name")
	if accountName == "" {
		return badRequest("Missing or invalid account_name")
	}
	callbackURL := req.r.FormValue("callback_url")
	if callbackURL == "" {
		return badRequest("Missing or invalid callback_url")
	}

	url, err := c.PipeService.GetAuthURL(serviceID, accountName, callbackURL)
	if err != nil {
		if errors.Is(err, &service.LoadError{}) {
			return badRequest(err.Error())
		}
		return internalServerError(err.Error())
	}
	return ok(struct {
		AuthURL string `json:"auth_url"`
	}{url})
}

func (c *Controller) CreateAuthorizationHandler(req Request) Response {
	currentToken := currentWorkspaceToken(req.r)
	workspaceID := currentWorkspaceID(req.r)
	serviceID, err := c.getServiceId(req)
	if err != nil {
		return badRequest(err.Error())
	}
	if len(req.body) == 0 {
		return badRequest("Missing payload")
	}

	var params domain.AuthParams
	err = json.Unmarshal(req.body, &params)
	if err != nil {
		return badRequest("Bad payload")
	}

	err = c.PipeService.CreateAuthorization(workspaceID, serviceID, currentToken, params)
	if err != nil {
		return internalServerError(err.Error())
	}
	return ok(nil)
}

func (c *Controller) DeleteAuthorizationHandler(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID, err := c.getServiceId(req)
	if err != nil {
		return badRequest(err.Error())
	}
	err = c.PipeService.DeleteAuthorization(workspaceID, serviceID)
	if err != nil {
		return internalServerError(err.Error())
	}

	return ok(nil)
}

func (c *Controller) GetServiceAccountsHandler(req Request) Response {
	forceImport := req.r.FormValue("force")
	workspaceID := currentWorkspaceID(req.r)
	serviceID, err := c.getServiceId(req)
	if err != nil {
		return badRequest(err.Error())
	}
	fi, err := strconv.ParseBool(forceImport)
	if err != nil {
		fi = false
	}

	accountsResponse, err := c.PipeService.GetServiceAccounts(workspaceID, serviceID, fi)
	if err != nil {
		if errors.Is(err, &service.LoadError{}) {
			return badRequest(err.Error())
		}
		if errors.Is(err, &service.RefreshError{}) {
			return badRequest(err.Error())
		}
		if errors.Is(err, service.ErrNoContent) {
			return noContent()
		}

		return internalServerError(err.Error())
	}

	return ok(accountsResponse)
}

func (c *Controller) GetServiceUsersHandler(req Request) Response {
	forceImport := req.r.FormValue("force")
	workspaceID := currentWorkspaceID(req.r)
	serviceID, err := c.getServiceId(req)
	if err != nil {
		return badRequest(err.Error())
	}

	fi, err := strconv.ParseBool(forceImport)
	if err != nil {
		fi = false
	}

	usersResponse, err := c.PipeService.GetServiceUsers(workspaceID, serviceID, fi)
	if err != nil {
		if errors.Is(err, service.ErrNoContent) {
			return noContent()
		}

		if errors.Is(err, &service.LoadError{}) {
			return badRequest("No authorizations for " + serviceID)
		}

		if errors.Is(err, service.ErrPipeNotConfigured) {
			return badRequest(err.Error())
		}

		if errors.Is(err, &service.SetParamsError{}) {
			return badRequest(err.Error())
		}
		return internalServerError(err.Error())
	}
	return ok(usersResponse)
}

func (c *Controller) GetServicePipeLogHandler(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID, pipeID := currentServicePipeID(req.r)
	pipesLog, err := c.PipeService.GetServicePipeLog(workspaceID, serviceID, pipeID)
	if err != nil {
		if errors.Is(err, service.ErrNoContent) {
			return noContent()
		}
		return internalServerError("Unable to get log from DB")
	}
	return Response{http.StatusOK, pipesLog, "text/plain"}
}

// PostServicePipeClearIDMappingsHandler clears mappings between pipes entities.
// TODO: Remove (Probably dead endpoint).
func (c *Controller) PostServicePipeClearIDMappingsHandler(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID, pipeID := currentServicePipeID(req.r)

	err := c.PipeService.ClearIDMappings(workspaceID, serviceID, pipeID)
	if err != nil {
		if errors.Is(err, service.ErrPipeNotConfigured) {
			return badRequest(err.Error())
		}
		return internalServerError("Unable to get clear connections: " + err.Error())
	}

	return noContent()
}

func (c *Controller) PostPipeRunHandler(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	serviceID, pipeID := currentServicePipeID(req.r)

	var selector domain.UserParams
	if pipeID == integration.UsersPipe {
		if err := json.Unmarshal(req.body, &selector); err != nil {
			return badRequest(fmt.Errorf("unable to parse users list, reason: %w", err))
		}
	}

	err := c.Queue.SchedulePipeSynchronization(workspaceID, serviceID, pipeID, selector)
	if err != nil {
		if errors.Is(err, service.ErrPipeNotConfigured) {
			return badRequest(err.Error())
		}
		if errors.Is(err, &service.SetParamsError{}) {
			return badRequest(err.Error())
		}
		return internalServerError(err.Error())
	}
	return ok(nil)
}

func (c *Controller) GetStatusHandler(Request) Response {
	resp := &struct {
		Reasons []string `json:"reasons"`
	}{}

	errs := c.PipeService.Ready()
	for _, err := range errs {
		resp.Reasons = append(resp.Reasons, err.Error())
	}

	if len(resp.Reasons) > 0 {
		return serviceUnavailable(resp)
	}
	return ok(map[string]string{
		"status":     "OK",
		"version":    c.Params.Version,
		"revision":   c.Params.Revision,
		"build_time": c.Params.BuildTime,
	})
}

func (c *Controller) getIntegrationParams(req Request) (integration.ID, integration.PipeID, error) {
	serviceID := integration.ID(mux.Vars(req.r)["service"])
	if !c.IntegrationsStorage.IsValidService(serviceID) {
		return "", "", errors.New("missing or invalid service")
	}
	pipeID := integration.PipeID(mux.Vars(req.r)["pipe"])
	if !c.IntegrationsStorage.IsValidPipe(pipeID) {
		return "", "", errors.New("Missing or invalid pipe")
	}
	return serviceID, pipeID, nil
}

func (c *Controller) getServiceId(req Request) (integration.ID, error) {
	serviceID := integration.ID(mux.Vars(req.r)["service"])
	if !c.IntegrationsStorage.IsValidService(serviceID) {
		return "", errors.New("missing or invalid service")
	}
	return serviceID, nil
}
