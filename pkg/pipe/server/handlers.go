package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"github.com/toggl/pipes-api/pkg/integrations"
	"github.com/toggl/pipes-api/pkg/pipe"
	"github.com/toggl/pipes-api/pkg/pipe/service"
)

type Controller struct {
	pipesSvc pipe.Service
}

func NewController(pipes pipe.Service) *Controller {
	return &Controller{pipesSvc: pipes}
}

func (c *Controller) GetIntegrationsHandler(req Request) Response {
	workspaceID := currentWorkspaceID(req.r)
	resp, err := c.pipesSvc.WorkspaceIntegrations(workspaceID)
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
	p, err := c.pipesSvc.GetPipe(workspaceID, serviceID, pipeID)
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
	err = c.pipesSvc.CreatePipe(workspaceID, serviceID, pipeID, req.body)
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

	err = c.pipesSvc.UpdatePipe(workspaceID, serviceID, pipeID, req.body)
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
	err = c.pipesSvc.DeletePipe(workspaceID, serviceID, pipeID)
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

	url, err := c.pipesSvc.GetAuthURL(serviceID, accountName, callbackURL)
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

	var params pipe.AuthParams
	err = json.Unmarshal(req.body, &params)
	if err != nil {
		return badRequest("Bad payload")
	}

	err = c.pipesSvc.CreateAuthorization(workspaceID, serviceID, currentToken, params)
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
	err = c.pipesSvc.DeleteAuthorization(workspaceID, serviceID)
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

	accountsResponse, err := c.pipesSvc.GetServiceAccounts(workspaceID, serviceID, fi)
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

	usersResponse, err := c.pipesSvc.GetServiceUsers(workspaceID, serviceID, fi)
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
	pipesLog, err := c.pipesSvc.GetServicePipeLog(workspaceID, serviceID, pipeID)
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

	err := c.pipesSvc.ClearIDMappings(workspaceID, serviceID, pipeID)
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

	err := c.pipesSvc.RunPipe(workspaceID, serviceID, pipeID, req.body)
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

	errs := c.pipesSvc.Ready()
	for _, err := range errs {
		resp.Reasons = append(resp.Reasons, err.Error())
	}

	if len(resp.Reasons) > 0 {
		return serviceUnavailable(resp)
	}
	return ok(map[string]string{"status": "OK"})
}

func (c *Controller) getIntegrationParams(req Request) (integrations.ExternalServiceID, integrations.PipeID, error) {
	serviceID := integrations.ExternalServiceID(mux.Vars(req.r)["service"])
	if !c.pipesSvc.AvailableServiceType(serviceID) {
		return "", "", errors.New("missing or invalid service")
	}
	pipeID := integrations.PipeID(mux.Vars(req.r)["pipe"])
	if !c.pipesSvc.AvailablePipeType(pipeID) {
		return "", "", errors.New("Missing or invalid pipe")
	}
	return serviceID, pipeID, nil
}

func (c *Controller) getServiceId(req Request) (integrations.ExternalServiceID, error) {
	serviceID := integrations.ExternalServiceID(mux.Vars(req.r)["service"])
	if !c.pipesSvc.AvailableServiceType(serviceID) {
		return "", errors.New("missing or invalid service")
	}
	return serviceID, nil
}
