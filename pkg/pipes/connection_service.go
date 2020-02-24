package pipes

import (
	"database/sql"
	"encoding/json"

	"github.com/toggl/pipes-api/pkg/integrations"
	"github.com/toggl/pipes-api/pkg/storage"
)

type ConnectionService struct {
	Storage *storage.Storage
}

func (cs *ConnectionService) Save(c *Connection) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	_, err = cs.Storage.Exec(insertConnectionSQL, c.workspaceID, c.key, b)
	if err != nil {
		return err
	}
	return nil
}

func (cs *ConnectionService) Load(rows *sql.Rows, c *Connection) error {
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

func (cs *ConnectionService) LoadConnection(s integrations.Service, pipeID string) (*Connection, error) {
	rows, err := cs.Storage.Query(selectConnectionSQL, s.GetWorkspaceID(), s.KeyFor(pipeID))
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

func (cs *ConnectionService) LoadConnectionRev(s integrations.Service, pipeID string) (*ReversedConnection, error) {
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
