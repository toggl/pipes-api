package main

import (
	"code.google.com/p/goauth2/oauth"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/tambet/oauthplain"
)

type Authorization struct {
	WorkspaceID    int
	ServiceID      string
	WorkspaceToken string
	Data           []byte
}

const (
	selectAuthorizationSQL = `SELECT
		workspace_id, service, workspace_token, data
		FROM authorizations
		WHERE workspace_id = $1
		AND service = $2
		LIMIT 1
  `
	insertAuthorizationSQL = `INSERT INTO
		authorizations(workspace_id, service, workspace_token, data)
		VALUES($1, $2, $3, $4)
  `
	deleteAuthorizationSQL = `DELETE FROM authorizations
		WHERE workspace_id = $1
		AND service = $2
	`
)

func NewAuthorization(workspaceID int, serviceID string) *Authorization {
	return &Authorization{
		WorkspaceID: workspaceID,
		ServiceID:   serviceID,
	}
}

func loadAuth(s Service) (*Authorization, error) {
	rows, err := db.Query(selectAuthorizationSQL, s.WorkspaceID(), s.Name())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, rows.Err()
	}
	var authorization Authorization
	if err := authorization.load(rows); err != nil {
		return nil, err
	}
	if err := s.setAuthData(authorization.Data); err != nil {
		return nil, err
	}
	return &authorization, nil
}

func (a *Authorization) save() error {
	_, err := db.Exec(insertAuthorizationSQL,
		a.WorkspaceID, a.ServiceID, a.WorkspaceToken, a.Data)
	if err != nil {
		return err
	}
	return nil
}

func (a *Authorization) load(rows *sql.Rows) error {
	err := rows.Scan(&a.WorkspaceID, &a.ServiceID, &a.WorkspaceToken, &a.Data)
	if err != nil {
		return err
	}
	return nil
}

func (a *Authorization) destroy(s Service) error {
	_, err := db.Exec(deleteAuthorizationSQL, s.WorkspaceID(), s.Name())
	return err
}

func loadAuthorizations(workspaceID int) (map[string]bool, error) {
	authorizations := make(map[string]bool)
	rows, err := db.Query(`
    SELECT service FROM authorizations
    WHERE workspace_id = $1`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var service string
		if err := rows.Scan(&service); err != nil {
			return nil, err
		}
		authorizations[service] = true
	}
	return authorizations, nil
}

func oAuth2URL(service string) string {
	config, ok := oAuth2Configs[service+"_"+*environment]
	if !ok {
		return ""
	}
	return config.AuthCodeURL("__STATE__") + "&type=web_server"
}

func oAuth1Exchange(serviceID string, payload map[string]interface{}) ([]byte, error) {
	accountName := payload["account_name"].(string)
	if accountName == "" {
		return nil, errors.New("missing account_name")
	}
	oAuthToken := payload["oauth_token"].(string)
	if oAuthToken == "" {
		return nil, errors.New("missing oauth_token")
	}
	oAuthVerifier := payload["oauth_verifier"].(string)
	if oAuthVerifier == "" {
		return nil, errors.New("missing oauth_verifier")
	}
	config, res := oAuth1Configs[serviceID]
	if !res {
		return nil, errors.New("service OAuth config not found")
	}
	transport := &oauthplain.Transport{
		Config: config.UpdateURLs(accountName),
	}
	token := &oauthplain.Token{
		OAuthToken:    oAuthToken,
		OAuthVerifier: oAuthVerifier,
	}
	if err := transport.Exchange(token); err != nil {
		return nil, err
	}
	token.Extra = make(map[string]string)
	token.Extra["account_name"] = accountName
	b, err := json.Marshal(token)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func oAuth2Exchange(serviceID string, payload map[string]interface{}) ([]byte, error) {
	code := payload["code"].(string)
	if code == "" {
		return nil, errors.New("missing code")
	}
	config, res := oAuth2Configs[serviceID+"_"+*environment]
	if !res {
		return nil, errors.New("service OAuth config not found")
	}
	transport := &oauth.Transport{Config: config}
	token, err := transport.Exchange(code)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(token)
	if err != nil {
		return nil, err
	}
	return b, nil
}
