package storage

import (
	"database/sql"
	"encoding/json"

	_ "github.com/lib/pq"

	"github.com/toggl/pipes-api/pkg/domain"
)

// IdMappingStorage SQL queries
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

type IdMappingStorage struct {
	db *sql.DB
}

func NewIdMappingStorage(db *sql.DB) *IdMappingStorage {
	if db == nil {
		panic("IdMappingStorage.db should not be nil")
	}
	return &IdMappingStorage{db: db}
}

func (ims *IdMappingStorage) Delete(workspaceID int, pipeConnectionKey, pipeStatusKey string) (err error) {
	tx, err := ims.db.Begin()
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

func (ims *IdMappingStorage) Save(c *domain.IDMapping) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	_, err = ims.db.Exec(insertConnectionSQL, c.WorkspaceID, c.Key, b)
	if err != nil {
		return err
	}
	return nil
}

func (ims *IdMappingStorage) Load(workspaceID int, key string) (*domain.IDMapping, error) {
	return ims.loadIDMapping(workspaceID, key)
}

func (ims *IdMappingStorage) LoadReversed(workspaceID int, key string) (*domain.ReversedIDMapping, error) {
	connection, err := ims.loadIDMapping(workspaceID, key)
	if err != nil {
		return nil, err
	}
	reversed := domain.NewReversedConnection()
	for key, value := range connection.Data {
		reversed.Data[value] = key
	}
	return reversed, nil
}

func (ims *IdMappingStorage) loadIDMapping(workspaceID int, key string) (*domain.IDMapping, error) {
	rows, err := ims.db.Query(selectConnectionSQL, workspaceID, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	connection := domain.NewIDMapping(workspaceID, key)
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
