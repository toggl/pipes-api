package main

import (
	"database/sql"
	"encoding/json"
)

const (
	selectConnectionSQL = `SELECT key, data
    FROM connections WHERE
    workspace_id = $1
    AND key = $2
    LIMIT 1
  `
	insertConnectionSQL = `
    WITH existing_connection AS (
      UPDATE connections SET data = $3
      WHERE workspace_id = $1 AND key = $2
      RETURNING key
    ),
    inserted_connection AS (
      INSERT INTO connections(workspace_id, key, data)
      SELECT $1, $2, $3
      WHERE NOT EXISTS (SELECT 1 FROM existing_connection)
      RETURNING key
    )
    SELECT * FROM inserted_connection
    UNION
    SELECT * FROM existing_connection
  `
)

type Connection struct {
	workspaceID int
	serviceID   string
	pipeID      string
	key         string
	Data        map[string]int
}

func NewConnection(s Service, pipeID string) *Connection {
	return &Connection{
		workspaceID: s.WorkspaceID(),
		key:         s.keyFor(pipeID),
		Data:        make(map[string]int),
	}
}

func loadConnection(s Service, pipeID string) (*Connection, error) {
	rows, err := db.Query(selectConnectionSQL, s.WorkspaceID(), s.keyFor(pipeID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	connection := Connection{Data: make(map[string]int)}
	if rows.Next() {
		if err := connection.load(rows); err != nil {
			return nil, err
		}
	}
	return &connection, nil
}

func (c *Connection) save() error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	_, err = db.Exec(insertConnectionSQL, c.workspaceID, c.key, b)
	if err != nil {
		return err
	}
	return nil
}

func (c *Connection) load(rows *sql.Rows) error {
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
