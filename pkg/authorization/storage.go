package authorization

import (
	"database/sql"
	"encoding/json"
	"errors"
	"sync"

	_ "github.com/lib/pq"

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
	truncateAuthorizationSQL = `TRUNCATE TABLE authorizations`
)

type OauthProvider interface {
	GetOAuth2Configs(externalServiceID string) (*oauth.Config, bool)
	Refresh(*oauth.Config, *oauth.Token) error
}

type Storage struct {
	db    *sql.DB
	oauth OauthProvider

	// Stores available authorization types for each service
	// Map format: map[externalServiceID]authType
	availableAuthTypes map[string]string
	mx                 sync.RWMutex
}

func NewStorage(db *sql.DB, oauth OauthProvider) *Storage {
	return &Storage{
		db:                 db,
		oauth:              oauth,
		availableAuthTypes: map[string]string{},
	}
}

func (as *Storage) GetAvailableAuthorizations(externalServiceID string) string {
	as.mx.RLock()
	defer as.mx.RUnlock()
	return as.availableAuthTypes[externalServiceID]
}

func (as *Storage) SetAuthorizationType(externalServiceID, authType string) {
	as.mx.Lock()
	defer as.mx.Unlock()
	as.availableAuthTypes[externalServiceID] = authType
}

func (as *Storage) Save(a *Authorization) error {
	_, err := as.db.Exec(insertAuthorizationSQL, a.WorkspaceID, a.ServiceID, a.WorkspaceToken, a.Data)
	if err != nil {
		return err
	}
	return nil
}

func (as *Storage) Load(workspaceID int, externalServiceID string) (*Authorization, error) {
	rows, err := as.db.Query(selectAuthorizationSQL, workspaceID, externalServiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, rows.Err()
	}
	var a Authorization
	err = rows.Scan(&a.WorkspaceID, &a.ServiceID, &a.WorkspaceToken, &a.Data)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (as *Storage) Destroy(workspaceID int, externalServiceID string) error {
	_, err := as.db.Exec(deleteAuthorizationSQL, workspaceID, externalServiceID)
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

	if err := as.oauth.Refresh(config, &token); err != nil {
		return err
	}
	err := a.SetOauth2Token(token)
	if err != nil {
		return err
	}
	if err := as.Save(a); err != nil {
		return err
	}
	return nil
}

// LoadWorkspaceAuthorizations loads map with authorizations status for each externalService.
// Map format: map[externalServiceID]isAuthorized
func (as *Storage) LoadWorkspaceAuthorizations(workspaceID int) (map[string]bool, error) {
	authorizations := make(map[string]bool)
	rows, err := as.db.Query(`SELECT service FROM authorizations WHERE workspace_id = $1`, workspaceID)
	if err != nil {
		return authorizations, err
	}
	defer rows.Close()
	for rows.Next() {
		var service string
		if err := rows.Scan(&service); err != nil {
			return authorizations, err
		}
		authorizations[service] = true
	}
	return authorizations, nil
}
