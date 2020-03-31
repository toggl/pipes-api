package service

import (
	"errors"

	"github.com/tambet/oauthplain"

	"github.com/toggl/pipes-api/pkg/domain"
)

type AuthorizationService struct {
	pipesStorage          domain.PipesStorage
	authorizationsStorage domain.AuthorizationsStorage
	integrationsStorage   domain.IntegrationsStorage
	oAuthProvider         domain.OAuthProvider
}

func NewAuthorizationService(pipesStorage domain.PipesStorage, authorizationsStorage domain.AuthorizationsStorage, integrationsStorage domain.IntegrationsStorage, oAuthProvider domain.OAuthProvider) *AuthorizationService {
	if pipesStorage == nil {
		panic("AuthorizationService.pipesStorage should not be nil")
	}
	if authorizationsStorage == nil {
		panic("AuthorizationService.authorizationsStorage should not be nil")
	}
	if integrationsStorage == nil {
		panic("AuthorizationService.integrationsStorage should not be nil")
	}
	if oAuthProvider == nil {
		panic("AuthorizationService.oAuthProvider should not be nil")
	}
	return &AuthorizationService{pipesStorage: pipesStorage, authorizationsStorage: authorizationsStorage, integrationsStorage: integrationsStorage, oAuthProvider: oAuthProvider}
}

func (svc *AuthorizationService) CreateAuthorization(workspaceID int, serviceID domain.IntegrationID, workspaceToken string, params domain.AuthParams) error {
	auth := domain.NewAuthorization(workspaceID, serviceID)
	auth.WorkspaceToken = workspaceToken

	authType, err := svc.integrationsStorage.LoadAuthorizationType(serviceID)
	if err != nil {
		return err
	}
	switch authType {
	case domain.TypeOauth1:
		token, err := svc.oAuthProvider.OAuth1Exchange(serviceID, params.AccountName, params.Token, params.Verifier)
		if err != nil {
			return err
		}
		if err := auth.SetOAuth1Token(token); err != nil {
			return err
		}
	case domain.TypeOauth2:
		if params.Code == "" {
			return errors.New("missing code")
		}
		token, err := svc.oAuthProvider.OAuth2Exchange(serviceID, params.Code)
		if err != nil {
			return err
		}
		if err := auth.SetOAuth2Token(token); err != nil {
			return err
		}
	}

	if err := svc.authorizationsStorage.Save(auth); err != nil {
		return err
	}
	return nil
}

func (svc *AuthorizationService) DeleteAuthorization(workspaceID int, serviceID domain.IntegrationID) error {
	if err := svc.authorizationsStorage.Delete(workspaceID, serviceID); err != nil {
		return err
	}
	if err := svc.pipesStorage.DeleteByWorkspaceIDServiceID(workspaceID, serviceID); err != nil {
		return err
	}
	return nil
}

func (svc *AuthorizationService) GetAuthURL(serviceID domain.IntegrationID, accountName, callbackURL string) (string, error) {
	config, found := svc.oAuthProvider.OAuth1Configs(serviceID)
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
