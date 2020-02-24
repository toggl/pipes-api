package storage

import (
	"database/sql"
	"encoding/json"

	"github.com/toggl/pipes-api/pkg/integrations"
)

type ConnectionStorage struct {
	db *sql.DB
}

func (cs *ConnectionStorage) Save(c *Connection) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	_, err = cs.db.Exec(insertConnectionSQL, c.workspaceID, c.key, b)
	if err != nil {
		return err
	}
	return nil
}

func (cs *ConnectionStorage) Load(rows *sql.Rows, c *Connection) error {
	var b []byte
	if err := rows.Scan(&c.key, &b); err != nil {
		return err
	}
	err := json.Unmarshal(b, c)
	if err != nil {
		return err
	}
	return nil
}

func (cs *ConnectionStorage) LoadConnection(s integrations.Integration, pipeID string) (*Connection, error) {
	rows, err := cs.db.Query(selectConnectionSQL, s.GetWorkspaceID(), s.KeyFor(pipeID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	connection := NewConnection(s, pipeID)
	if rows.Next() {
		if err := cs.Load(rows, connection); err != nil {
			return nil, err
		}
	}
	return connection, nil
}

func (cs *ConnectionStorage) LoadConnectionRev(s integrations.Integration, pipeID string) (*ReversedConnection, error) {
	connection, err := cs.LoadConnection(s, pipeID)
	if err != nil {
		return nil, err
	}
	reversed := &ReversedConnection{make(map[int]string)}
	for key, value := range connection.Data {
		reversed.Data[value] = key
	}
	return reversed, nil
}

const (
	selectConnectionSQL = `SELECT Key, data
    FROM connections WHERE
    workspace_id = $1
    AND Key = $2
    LIMIT 1
  `
	insertConnectionSQL = `
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
)
