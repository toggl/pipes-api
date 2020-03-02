package connection

import (
	"database/sql"
	"encoding/json"
)

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

type Storage struct {
	db *sql.DB
}

func NewStorage(db *sql.DB) *Storage {
	return &Storage{db: db}
}

func (cs *Storage) Save(c *Connection) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	_, err = cs.db.Exec(insertConnectionSQL, c.WorkspaceID, c.Key, b)
	if err != nil {
		return err
	}
	return nil
}

func (cs *Storage) Load(workspaceID int, key string) (*Connection, error) {
	return cs.load(workspaceID, key)
}

func (cs *Storage) LoadReversed(workspaceID int, key string) (*ReversedConnection, error) {
	connection, err := cs.load(workspaceID, key)
	if err != nil {
		return nil, err
	}
	reversed := &ReversedConnection{make(map[int]string)}
	for key, value := range connection.Data {
		reversed.Data[value] = key
	}
	return reversed, nil
}

func (cs *Storage) load(workspaceID int, key string) (*Connection, error) {
	rows, err := cs.db.Query(selectConnectionSQL, workspaceID, key)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	connection := NewConnection(workspaceID, key)
	if rows.Next() {
		if err := cs.scan(rows, connection); err != nil {
			return nil, err
		}
	}
	return connection, nil
}

func (cs *Storage) scan(rows *sql.Rows, c *Connection) error {
	var b []byte
	if err := rows.Scan(&c.Key, &b); err != nil {
		return err
	}
	err := json.Unmarshal(b, c)
	if err != nil {
		return err
	}
	return nil
}
