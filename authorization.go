package main

import (
	"database/sql"
	"encoding/json"
	"time"
)

type Authorization struct {
	AccessToken    string
	RefreshToken   string
	WorkspaceToken string
	Expiry         time.Time
}

const (
	selectAuthorizationSQL = `SELECT data
    FROM authorizations
    WHERE workspace_id = $1
    AND service = $2
    LIMIT 1
  `
	insertAuthorizationSQL = `INSERT INTO
		authorizations(workspace_id, service, data)
		VALUES($1, $2, $3)
  `
)

func serviceNotAuthorized(s Service) bool {
	if _, err := loadAuth(s); err != nil {
		return true
	}
	return false
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
	var auth Authorization
	if err := auth.load(rows); err != nil {
		return nil, err
	}
	s.setAuthData(&auth)
	return &auth, nil
}

func (a *Authorization) save(workspaceID int, serviceID string) error {
	b, err := json.Marshal(a)
	if err != nil {
		return err
	}
	_, err = db.Exec(insertAuthorizationSQL, workspaceID, serviceID, b)
	if err != nil {
		return err
	}
	return nil
}

func (a *Authorization) load(rows *sql.Rows) error {
	var b []byte
	if err := rows.Scan(&b); err != nil {
		return err
	}
	err := json.Unmarshal(b, a)
	if err != nil {
		return err
	}
	return nil
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

func getAuthURL(service string) string {
	config, ok := knownOauthConfigs[service+"_"+*environment]
	if !ok {
		panic("Oauth config not found!")
	}
	return config.AuthCodeURL("__STATE__") + "&type=web_server"
}
