package storage

import (
	"database/sql"
	"encoding/json"
	"errors"

	"code.google.com/p/goauth2/oauth"

	"github.com/toggl/pipes-api/pkg/environment"
	"github.com/toggl/pipes-api/pkg/integrations"
)

type AuthorizationStorage struct {
	db  *sql.DB
	env *environment.Environment
}

func (as *AuthorizationStorage) Save(a *environment.AuthorizationConfig) error {
	_, err := as.db.Exec(insertAuthorizationSQL,
		a.WorkspaceID, a.ServiceID, a.WorkspaceToken, a.Data)
	if err != nil {
		return err
	}
	return nil
}

func (as *AuthorizationStorage) Load(rows *sql.Rows, a *environment.AuthorizationConfig) error {
	err := rows.Scan(&a.WorkspaceID, &a.ServiceID, &a.WorkspaceToken, &a.Data)
	if err != nil {
		return err
	}
	return nil
}

func (as *AuthorizationStorage) Destroy(s integrations.Integration) error {
	_, err := as.db.Exec(deleteAuthorizationSQL, s.GetWorkspaceID(), s.Name())
	return err
}

func (as *AuthorizationStorage) Refresh(a *environment.AuthorizationConfig) error {
	if as.env.GetAvailableAuthorizations(a.ServiceID) != "oauth2" { // TODO: Remove global state.
		return nil
	}
	var token oauth.Token
	if err := json.Unmarshal(a.Data, &token); err != nil {
		return err
	}
	if !token.Expired() {
		return nil
	}
	config, res := as.env.GetOAuth2Configs(a.ServiceID)
	if !res {
		return errors.New("service OAuth config not found")
	}
	transport := &oauth.Transport{Config: config, Token: &token}
	if err := transport.Refresh(); err != nil {
		return err
	}
	b, err := json.Marshal(token)
	if err != nil {
		return err
	}
	a.Data = b
	if err := as.Save(a); err != nil {
		return err
	}
	return nil
}

func (as *AuthorizationStorage) LoadAuth(s integrations.Integration) (*environment.AuthorizationConfig, error) {
	rows, err := as.db.Query(selectAuthorizationSQL, s.GetWorkspaceID(), s.Name())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, rows.Err()
	}
	var authorization environment.AuthorizationConfig
	if err := as.Load(rows, &authorization); err != nil {
		return nil, err
	}
	if err := s.SetAuthData(authorization.Data); err != nil {
		return nil, err
	}
	return &authorization, nil
}

func (as *AuthorizationStorage) LoadAuthorizations(workspaceID int) (map[string]bool, error) {
	authorizations := make(map[string]bool)
	rows, err := as.db.Query(`SELECT service FROM authorizations WHERE workspace_id = $1`, workspaceID)
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

const (
	selectAuthorizationSQL = `SELECT
		workspace_id, service, workspace_token, data
		FROM authorizations
		WHERE workspace_id = $1
		AND service = $2
		LIMIT 1
  `
	insertAuthorizationSQL = `WITH existing_auth AS (
		UPDATE authorizations SET data = $4, workspace_token = $3
		WHERE workspace_id = $1 AND service = $2
		RETURNING service
	),
	inserted_auth AS (
		INSERT INTO
		authorizations(workspace_id, service, workspace_token, data)
		SELECT $1, $2, $3, $4
		WHERE NOT EXISTS (SELECT 1 FROM existing_auth)
		RETURNING service
	)
	SELECT * FROM inserted_auth
	UNION
	SELECT * FROM existing_auth
  `
	deleteAuthorizationSQL = `DELETE FROM authorizations
		WHERE workspace_id = $1
		AND service = $2
	`
)
