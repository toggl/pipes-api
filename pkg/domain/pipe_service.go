package domain

import (
	"encoding/json"
	"errors"
	"log"

	"github.com/tambet/oauthplain"

	"github.com/toggl/pipes-api/pkg/integration"
	"github.com/toggl/pipes-api/pkg/toggl"
)

type Service struct {
	*AuthorizationFactory
	*PipeFactory

	PipesStorage
	AuthorizationsStorage
	IntegrationsStorage
	IDMappingsStorage
	ImportsStorage

	OAuthProvider
	TogglClient
}

func (svc *Service) Ready() []error {
	errs := make([]error, 0)

	if svc.PipesStorage.IsDown() {
		errs = append(errs, errors.New("database is down"))
	}

	if err := svc.TogglClient.Ping(); err != nil {
		errs = append(errs, err)
	}
	return errs
}

func (svc *Service) GetPipe(workspaceID int, serviceID integration.ID, pipeID integration.PipeID) (*Pipe, error) {
	p := svc.PipeFactory.Create(workspaceID, serviceID, pipeID)
	if err := svc.PipesStorage.Load(p); err != nil {
		return nil, err
	}
	var err error
	p.PipeStatus, err = svc.PipesStorage.LoadStatus(workspaceID, serviceID, pipeID)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (svc *Service) CreatePipe(workspaceID int, serviceID integration.ID, pipeID integration.PipeID, params []byte) error {
	p := svc.PipeFactory.Create(workspaceID, serviceID, pipeID)

	service := NewExternalService(serviceID, workspaceID)
	err := service.SetParams(params)
	if err != nil {
		return SetParamsError{err}
	}
	p.ServiceParams = params

	if err := svc.PipesStorage.Save(p); err != nil {
		return err
	}
	return nil
}

func (svc *Service) UpdatePipe(workspaceID int, serviceID integration.ID, pipeID integration.PipeID, params []byte) error {
	p := svc.PipeFactory.Create(workspaceID, serviceID, pipeID)
	if err := svc.PipesStorage.Load(p); err != nil {
		return err
	}
	if !p.Configured {
		return ErrPipeNotConfigured
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return err
	}
	if err := svc.PipesStorage.Save(p); err != nil {
		return err
	}
	return nil
}

func (svc *Service) DeletePipe(workspaceID int, serviceID integration.ID, pipeID integration.PipeID) error {
	p := svc.PipeFactory.Create(workspaceID, serviceID, pipeID)
	if err := svc.PipesStorage.Load(p); err != nil {
		return err
	}
	if err := svc.PipesStorage.Delete(p, workspaceID); err != nil {
		return err
	}
	return nil
}

var ErrNoContent = errors.New("no content")

func (svc *Service) GetServicePipeLog(workspaceID int, serviceID integration.ID, pipeID integration.PipeID) (string, error) {
	pipeStatus, err := svc.PipesStorage.LoadStatus(workspaceID, serviceID, pipeID)
	if err != nil {
		return "", err
	}
	if pipeStatus == nil {
		return "", ErrNoContent
	}
	return pipeStatus.GenerateLog(), nil
}

// TODO: Remove (Probably dead method).
func (svc *Service) ClearIDMappings(workspaceID int, serviceID integration.ID, pipeID integration.PipeID) error {
	p := svc.PipeFactory.Create(workspaceID, serviceID, pipeID)
	if err := svc.PipesStorage.Load(p); err != nil {
		return err
	}
	if !p.Configured {
		return ErrPipeNotConfigured
	}
	service := NewExternalService(p.ServiceID, p.WorkspaceID)
	if err := service.SetParams(p.ServiceParams); err != nil {
		return err
	}
	auth := svc.AuthorizationFactory.Create(workspaceID, serviceID)
	if err := svc.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth); err != nil {
		return err
	}
	if err := auth.Refresh(); err != nil {
		return err
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return err
	}
	pipeStatus, err := svc.PipesStorage.LoadStatus(p.WorkspaceID, p.ServiceID, p.ID)
	if err != nil {
		return err
	}

	err = svc.IDMappingsStorage.Delete(p.WorkspaceID, service.KeyFor(p.ID), pipeStatus.Key)
	if err != nil {
		return err
	}
	return nil
}

func (svc *Service) GetServiceUsers(workspaceID int, serviceID integration.ID, forceImport bool) (*toggl.UsersResponse, error) {
	service := NewExternalService(serviceID, workspaceID)
	auth := svc.AuthorizationFactory.Create(workspaceID, serviceID)
	err := svc.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth)
	if err != nil {
		return nil, LoadError{err}
	}
	if err := auth.Refresh(); err != nil {
		return nil, RefreshError{errors.New("oAuth refresh failed!")}
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return nil, err
	}

	usersPipe := svc.PipeFactory.Create(workspaceID, serviceID, integration.UsersPipe)
	if err := svc.PipesStorage.Load(usersPipe); err != nil {
		return nil, err
	}
	if usersPipe == nil {
		return nil, ErrPipeNotConfigured
	}
	if err := service.SetParams(usersPipe.ServiceParams); err != nil {
		return nil, SetParamsError{err}
	}

	if forceImport {
		if err := svc.ImportsStorage.DeleteUsersFor(service); err != nil {
			return nil, err
		}
	}

	usersResponse, err := svc.ImportsStorage.LoadUsersFor(service)
	if err != nil {
		return nil, err
	}

	if usersResponse == nil {
		if forceImport {
			go func() {
				fetchErr := usersPipe.FetchUsers()
				if fetchErr != nil {
					log.Print(fetchErr.Error())
				}
			}()
		}
		return nil, ErrNoContent
	}
	return usersResponse, nil
}

func (svc *Service) GetServiceAccounts(workspaceID int, serviceID integration.ID, forceImport bool) (*toggl.AccountsResponse, error) {
	service := NewExternalService(serviceID, workspaceID)
	auth := svc.AuthorizationFactory.Create(workspaceID, serviceID)
	err := svc.AuthorizationsStorage.Load(service.GetWorkspaceID(), service.ID(), auth)
	if err != nil {
		return nil, LoadError{err}
	}
	if err := auth.Refresh(); err != nil {
		return nil, RefreshError{errors.New("oAuth refresh failed!")}
	}
	if err := service.SetAuthData(auth.Data); err != nil {
		return nil, err
	}
	if forceImport {
		if err := svc.ImportsStorage.DeleteAccountsFor(service); err != nil {
			return nil, err
		}
	}

	accountsResponse, err := svc.ImportsStorage.LoadAccountsFor(service)
	if err != nil {
		return nil, err
	}

	if accountsResponse == nil {
		go func() {
			var response toggl.AccountsResponse
			accounts, err := service.Accounts()
			response.Accounts = accounts
			if err != nil {
				response.Error = err.Error()
			}
			if err := svc.ImportsStorage.SaveAccountsFor(service, response); err != nil {
				log.Print(err.Error())
			}
		}()
		return nil, ErrNoContent
	}
	return accountsResponse, nil
}

func (svc *Service) GetAuthURL(serviceID integration.ID, accountName, callbackURL string) (string, error) {
	config, found := svc.OAuthProvider.OAuth1Configs(serviceID)
	if !found {
		return "", LoadError{errors.New("env OAuth config not found")}
	}
	transport := &oauthplain.Transport{
		Config: config.UpdateURLs(accountName),
	}
	token, err := transport.AuthCodeURL(callbackURL)
	if err != nil {
		return "", err
	}
	return token.AuthorizeUrl, nil
}

func (svc *Service) CreateAuthorization(workspaceID int, serviceID integration.ID, workspaceToken string, params AuthParams) error {
	auth := svc.AuthorizationFactory.Create(workspaceID, serviceID)
	auth.WorkspaceToken = workspaceToken

	authType, err := svc.IntegrationsStorage.LoadAuthorizationType(serviceID)
	if err != nil {
		return err
	}
	switch authType {
	case TypeOauth1:
		token, err := svc.OAuthProvider.OAuth1Exchange(serviceID, params.AccountName, params.Token, params.Verifier)
		if err != nil {
			return err
		}
		if err := auth.SetOAuth1Token(token); err != nil {
			return err
		}
	case TypeOauth2:
		if params.Code == "" {
			return errors.New("missing code")
		}
		token, err := svc.OAuthProvider.OAuth2Exchange(serviceID, params.Code)
		if err != nil {
			return err
		}
		if err := auth.SetOAuth2Token(token); err != nil {
			return err
		}
	}

	if err := svc.AuthorizationsStorage.Save(auth); err != nil {
		return err
	}
	return nil
}

func (svc *Service) DeleteAuthorization(workspaceID int, serviceID integration.ID) error {
	if err := svc.AuthorizationsStorage.Delete(workspaceID, serviceID); err != nil {
		return err
	}
	if err := svc.PipesStorage.DeleteByWorkspaceIDServiceID(workspaceID, serviceID); err != nil {
		return err
	}
	return nil
}

func (svc *Service) GetIntegrations(workspaceID int) ([]Integration, error) {
	authorizations, err := svc.AuthorizationsStorage.LoadWorkspaceAuthorizations(workspaceID)
	if err != nil {
		return nil, err
	}
	workspacePipes, err := svc.PipesStorage.LoadAll(workspaceID)
	if err != nil {
		return nil, err
	}
	pipeStatuses, err := svc.PipesStorage.LoadAllStatuses(workspaceID)
	if err != nil {
		return nil, err
	}

	availableIntegrations, err := svc.IntegrationsStorage.LoadIntegrations()
	if err != nil {
		return nil, err
	}

	var resultIntegrations []Integration
	for _, current := range availableIntegrations {

		var ci = current
		ci.AuthURL = svc.OAuthProvider.OAuth2URL(ci.ID)
		ci.Authorized = authorizations[ci.ID]

		var pipes []*Pipe
		for i := range ci.Pipes {

			var p = *ci.Pipes[i]
			key := PipesKey(ci.ID, p.ID)
			var existing = workspacePipes[key]

			if existing != nil {
				p.Automatic = existing.Automatic
				p.Configured = existing.Configured
			}

			p.PipeStatus = pipeStatuses[key]
			pipes = append(pipes, &p)
		}
		ci.Pipes = pipes
		resultIntegrations = append(resultIntegrations, *ci)
	}
	return resultIntegrations, nil
}
