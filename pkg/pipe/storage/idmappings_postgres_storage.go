package storage

import (
	"database/sql"
	"encoding/json"

	_ "github.com/lib/pq"

	"github.com/toggl/pipes-api/pkg/pipe"
)

// PostgresStorage SQL queries
const (
	deletePipeConnectionsSQL = `DELETE FROM connections WHERE workspace_id = $1 AND Key = $2`
	selectConnectionSQL      = `SELECT Key, data FROM connections WHERE workspace_id = $1 AND Key = $2 LIMIT 1`
	insertConnectionSQL      = `
    WITH existing_connection AS (
      UPDATE connections SET data = $3
      WHERE workspace_id = $1 AND Key = $2
      RETURNING Key
    ),
    inserted_connection AS (
      INSERT INTO connections(workspace_id, Key, data)
      SELECT $1, $2, $3
      WHERE NOT EXISTS (SELECT 1 FROM existing_connection)
      RETURNING Key
    )
    SELECT * FROM inserted_connection
    UNION
    SELECT * FROM existing_connection
  `
	truncateConnectionSQL = `TRUNCATE TABLE connections`
)

type IDMappingsPostgresStorage struct {
	db *sql.DB
}

func NewIDMappingsPostgresStorage(db *sql.DB) *IDMappingsPostgresStorage {
	return &IDMappingsPostgresStorage{db: db}
}

func (ps *IDMappingsPostgresStorage) Delete(workspaceID int, pipeConnectionKey, pipeStatusKey string) (err error) {
	tx, err := ps.db.Begin()
	if err != nil {
		return
	}
	_, err = tx.Exec(deletePipeConnectionsSQL, workspaceID, pipeConnectionKey)
	if err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			err = rollbackErr
		}
		return
	}
	_, err = tx.Exec(deletePipeStatusSQL, workspaceID, pipeStatusKey)
	if err != nil {
		rollbackErr := tx.Rollback()
		if rollbackErr != nil {
			err = rollbackErr
		}

	}
	return tx.Commit()
}

func (ps *IDMappingsPostgresStorage) Save(c *pipe.IDMapping) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	_, err = ps.db.Exec(insertConnectionSQL, c.WorkspaceID, c.Key, b)
	if err != nil {
		return err
	}
	return nil
}

func (ps *IDMappingsPostgresStorage) Load(workspaceID int, key string) (*pipe.IDMapping, error) {
	return ps.loadIDMapping(workspaceID, key)
}

func (ps *IDMappingsPostgresStorage) LoadReversed(workspaceID int, key string) (*pipe.ReversedIDMapping, error) {
	connection, err := ps.loadIDMapping(workspaceID, key)
	if err != nil {
		return nil, err
	}
	reversed := pipe.NewReversedConnection()
	for key, value := range connection.Data {
		reversed.Data[value] = key
	}
	return reversed, nil
}

func (ps *IDMappingsPostgresStorage) loadIDMapping(workspaceID int, key string) (*pipe.IDMapping, error) {
	rows, err := ps.db.Query(selectConnectionSQL, workspaceID, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	connection := pipe.NewIDMapping(workspaceID, key)
	if rows.Next() {
		var b []byte
		if err := rows.Scan(&connection.Key, &b); err != nil {
			return nil, err
		}
		err := json.Unmarshal(b, connection)
		if err != nil {
			return nil, err
		}
	}
	return connection, nil
}
