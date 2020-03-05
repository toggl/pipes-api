package storage

const (
	selectPipesSQL = `SELECT workspace_id, Key, data
    FROM pipes WHERE workspace_id = $1
  `
	singlePipesSQL = `SELECT workspace_id, Key, data
    FROM pipes WHERE workspace_id = $1
    AND Key = $2 LIMIT 1
  `
	deletePipeSQL = `DELETE FROM pipes
    WHERE workspace_id = $1
    AND Key LIKE $2
  `
	insertPipesSQL = `
    WITH existing_pipe AS (
      UPDATE pipes SET data = $3
      WHERE workspace_id = $1 AND Key = $2
      RETURNING Key
    ),
    inserted_pipe AS (
      INSERT INTO pipes(workspace_id, Key, data)
      SELECT $1, $2, $3
      WHERE NOT EXISTS (SELECT 1 FROM existing_pipe)
      RETURNING Key
    )
    SELECT * FROM inserted_pipe
    UNION
    SELECT * FROM existing_pipe
  `
	deletePipeConnectionsSQL = `DELETE FROM connections
    WHERE workspace_id = $1
    AND Key = $2
  `
	selectPipesFromQueueSQL = `SELECT workspace_id, Key
	FROM get_queued_pipes()`

	queueAutomaticPipesSQL = `SELECT queue_automatic_pipes()`

	queuePipeAsFirstSQL = `SELECT queue_pipe_as_first($1, $2)`

	setQueuedPipeSyncedSQL = `UPDATE queued_pipes
	SET synced_at = now()
	WHERE workspace_id = $1
	AND Key = $2
	AND locked_at IS NOT NULL
	AND synced_at IS NULL`
	truncatePipesSQL = `TRUNCATE TABLE pipes`
)

const (
	selectPipeStatusSQL = `SELECT Key, data
    FROM pipes_status
    WHERE workspace_id = $1
  `
	singlePipeStatusSQL = `SELECT data
    FROM pipes_status
    WHERE workspace_id = $1
    AND Key = $2 LIMIT 1
  `
	deletePipeStatusSQL = `DELETE FROM pipes_status
		WHERE workspace_id = $1
		AND Key LIKE $2
  `
	lastSyncSQL = `SELECT (data->>'sync_date')::timestamp with time zone
    FROM pipes_status
    WHERE workspace_id = $1
    AND Key = $2
  `
	insertPipeStatusSQL = `
    WITH existing_status AS (
      UPDATE pipes_status SET data = $3
      WHERE workspace_id = $1 AND Key = $2
      RETURNING Key
    ),
    inserted_status AS (
      INSERT INTO pipes_status(workspace_id, Key, data)
      SELECT $1, $2, $3
      WHERE NOT EXISTS (SELECT 1 FROM existing_status)
      RETURNING Key
    )
    SELECT * FROM inserted_status
    UNION
    SELECT * FROM existing_status
  `
	truncatePipesStatusSQL = `TRUNCATE TABLE pipes_status`
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
	truncateConnectionSQL = `TRUNCATE TABLE connections`
)

const (
	truncateImportsSQL = `TRUNCATE TABLE imports`
)

const (
	truncateQueuedPipesSQL = `TRUNCATE TABLE queued_pipes`
)
