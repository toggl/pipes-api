package authorization

import (
	"database/sql"
	"encoding/json"
	"errors"
	"sync"

	"code.google.com/p/goauth2/oauth"
)

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

type OauthProvider interface {
	GetOAuth2Configs(serviceID string) (*oauth.Config, bool)
}

type Storage struct {
	db    *sql.DB
	oauth OauthProvider

	availableAuthorizations map[string]string
	mx                      sync.RWMutex
}

func NewStorage(db *sql.DB, oauth OauthProvider) *Storage {
	return &Storage{
		db:                      db,
		oauth:                   oauth,
		availableAuthorizations: map[string]string{},
	}
}

func (as *Storage) Save(a *Authorization) error {
	_, err := as.db.Exec(insertAuthorizationSQL, a.WorkspaceID, a.ServiceID, a.WorkspaceToken, a.Data)
	if err != nil {
		return err
	}
	return nil
}

func (as *Storage) GetAvailableAuthorizations(serviceID string) string {
	as.mx.RLock()
	defer as.mx.RUnlock()
	return as.availableAuthorizations[serviceID]
}

func (as *Storage) SetAuthorizationType(integrationID, authType string) {
	as.mx.Lock()
	defer as.mx.Unlock()
	as.availableAuthorizations[integrationID] = authType
}

func (as *Storage) Load(rows *sql.Rows, a *Authorization) error {
	err := rows.Scan(&a.WorkspaceID, &a.ServiceID, &a.WorkspaceToken, &a.Data)
	if err != nil {
		return err
	}
	return nil
}

func (as *Storage) Destroy(workspaceID int, externalServiceName string) error {
	_, err := as.db.Exec(deleteAuthorizationSQL, workspaceID, externalServiceName)
	return err
}

func (as *Storage) Refresh(a *Authorization) error {
	if as.GetAvailableAuthorizations(a.ServiceID) != TypeOauth2 {
		return nil
	}
	var token oauth.Token
	if err := json.Unmarshal(a.Data, &token); err != nil {
		return err
	}
	if !token.Expired() {
		return nil
	}
	config, res := as.oauth.GetOAuth2Configs(a.ServiceID)
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

func (as *Storage) LoadAuth(workspaceID int, externalServiceName string) (*Authorization, error) {
	rows, err := as.db.Query(selectAuthorizationSQL, workspaceID, externalServiceName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, rows.Err()
	}
	var authorization Authorization
	if err := as.Load(rows, &authorization); err != nil {
		return nil, err
	}
	return &authorization, nil
}

func (as *Storage) LoadAuthorizations(workspaceID int) (map[string]bool, error) {
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
