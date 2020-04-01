package storage

import (
	"database/sql"

	_ "github.com/lib/pq"

	"github.com/toggl/pipes-api/pkg/domain"
)

// AuthorizationStorage SQL queries
const (
	selectAuthorizationSQL = `SELECT workspace_id, service, workspace_token, data
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
	deleteAuthorizationSQL   = `DELETE FROM authorizations WHERE workspace_id = $1 AND service = $2`
	truncateAuthorizationSQL = `TRUNCATE TABLE authorizations`
)

type AuthorizationStorage struct {
	db *sql.DB
}

func NewAuthorizationStorage(db *sql.DB) *AuthorizationStorage {
	if db == nil {
		panic("AuthorizationStorage.db should not be nil")
	}
	return &AuthorizationStorage{db: db}
}

func (as *AuthorizationStorage) Save(a *domain.Authorization) error {
	_, err := as.db.Exec(insertAuthorizationSQL, a.WorkspaceID, a.ServiceID, a.WorkspaceToken, a.Data)
	if err != nil {
		return err
	}
	return nil
}

func (as *AuthorizationStorage) Load(workspaceID int, externalServiceID domain.IntegrationID, a *domain.Authorization) error {
	rows, err := as.db.Query(selectAuthorizationSQL, workspaceID, externalServiceID)
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		return rows.Err()
	}
	err = rows.Scan(&a.WorkspaceID, &a.ServiceID, &a.WorkspaceToken, &a.Data)
	if err != nil {
		return err
	}
	return nil
}

func (as *AuthorizationStorage) Delete(workspaceID int, externalServiceID domain.IntegrationID) error {
	_, err := as.db.Exec(deleteAuthorizationSQL, workspaceID, externalServiceID)
	return err
}

// LoadWorkspaceAuthorizations loads map with authorizations status for each externalService.
// Map format: map[externalServiceID]isAuthorized
func (as *AuthorizationStorage) LoadWorkspaceAuthorizations(workspaceID int) (map[domain.IntegrationID]bool, error) {
	authorizations := make(map[domain.IntegrationID]bool)
	rows, err := as.db.Query(`SELECT service FROM authorizations WHERE workspace_id = $1`, workspaceID)
	if err != nil {
		return authorizations, err
	}
	defer rows.Close()
	for rows.Next() {
		var service domain.IntegrationID
		if err := rows.Scan(&service); err != nil {
			return authorizations, err
		}
		authorizations[service] = true
	}
	return authorizations, nil
}
