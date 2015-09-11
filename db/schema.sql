CREATE ROLE pipes_user WITH LOGIN;

CREATE TABLE authorizations(
  workspace_id INTEGER,
  workspace_token VARCHAR(50),
  service VARCHAR(50),
  data JSON
);

CREATE TABLE imports(
  workspace_id INTEGER,
  key VARCHAR(50),
  data JSON,
  created_at TIMESTAMP
);

CREATE TABLE pipes(
  workspace_id INTEGER,
  key VARCHAR(50),
  data JSON
);

CREATE TABLE pipes_status(
  workspace_id INTEGER,
  key VARCHAR(50),
  data JSON
);

CREATE TABLE connections(
  workspace_id INTEGER,
  key VARCHAR(50),
  data JSON
);

ALTER TABLE pipes ADD CONSTRAINT pipes_pk PRIMARY KEY (workspace_id, key);

CREATE TABLE queued_pipes (
  workspace_id INTEGER,
  key VARCHAR(50),
  priority INTEGER DEFAULT 0,
  created_at timestamp without time zone DEFAULT now(),
  locked_at timestamp without time zone DEFAULT NULL,
  synced_at timestamp without time zone DEFAULT NULL,
  FOREIGN KEY (workspace_id, key) REFERENCES pipes (workspace_id, key) ON DELETE CASCADE
);
  
CREATE UNIQUE INDEX pipes_queue_unique ON queued_pipes (workspace_id, key, coalesce(locked_at, '0001-01-01 00:00:00'::timestamp), coalesce(synced_at,'0001-01-01 00:00:00'::timestamp));

CREATE OR REPLACE FUNCTION get_queued_pipes() RETURNS TABLE(workspace_id INTEGER, key VARCHAR(50)) AS $$
BEGIN
  RETURN QUERY
  UPDATE
    queued_pipes
  SET 
    locked_at = NOW()
  FROM (
    SELECT queued_pipes.workspace_id, queued_pipes.key
    FROM queued_pipes
    WHERE locked_at IS NULL
    AND synced_at IS NULL
    ORDER BY priority desc, created_at asc
    FOR UPDATE
    LIMIT 10
    ) as pipe
  WHERE pipe.workspace_id = queued_pipes.workspace_id AND pipe.key = queued_pipes.key AND synced_at IS NULL
  RETURNING pipe.workspace_id, pipe.key;
END;
$$
LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION queue_automatic_pipes() RETURNS VOID AS $$
DECLARE
  r pipes%rowtype;
BEGIN
  FOR r IN SELECT workspace_id, key FROM pipes
  WHERE data->>'automatic' = 'true'
  LOOP
  INSERT INTO queued_pipes (workspace_id, key)
  VALUES (r.workspace_id, r.key)
  WHERE NOT EXISTS
  (
    SELECT 1 FROM queued_pipes 
    WHERE workspace_id = r.workspace_id 
    AND key = r.key 
    AND synced_at IS NULL 
    FOR UPDATE
  );
  END LOOP;
END;
$$
LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION queue_pipe_as_first(workspace_id_param INTEGER, key_param VARCHAR(50)) RETURNS VOID AS $$
BEGIN
  WITH priority_cte AS (
    SELECT max(priority)+1 as new_priority FROM queued_pipes WHERE locked_at IS NULL AND synced_at IS NULL
  ),
  existing_pipe AS
  (
    UPDATE queued_pipes
    SET priority = new_priority FROM priority_cte
    WHERE workspace_id = workspace_id_param 
    AND key = key_param 
    AND locked_at IS NULL 
    AND synced_at IS NULL
    RETURNING workspace_id
  )
  INSERT INTO queued_pipes (workspace_id, key, priority)
  SELECT workspace_id_param, key_param, new_priority FROM priority_cte
  WHERE NOT EXISTS (SELECT 1 FROM existing_pipe)
  AND NOT EXISTS
  (
    SELECT 1 FROM queued_pipes 
    WHERE workspace_id = workspace_id_param 
    AND key = key_param 
    AND synced_at IS NULL 
    FOR UPDATE
  );
END;
$$
LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION remove_locked_from_queue(age INTERVAL) RETURNS VOID AS $$
BEGIN
  DELETE FROM queued_pipes
  WHERE synced_at IS NULL
  AND locked_at < (now() - age);
END;
$$
LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION remove_synced_from_queue(age INTERVAL) RETURNS VOID AS $$
BEGIN
  DELETE FROM queued_pipes
  WHERE synced_at < (now() - age);
END;
$$
LANGUAGE plpgsql;

ALTER TABLE authorizations OWNER TO pipes_user;
ALTER TABLE imports OWNER TO pipes_user;
ALTER TABLE pipes OWNER TO pipes_user;
ALTER TABLE pipes_status OWNER TO pipes_user;
ALTER TABLE connections OWNER TO pipes_user;
ALTER TABLE queued_pipes OWNER TO pipes_user;

ALTER FUNCTION get_queued_pipes() OWNER TO pipes_user;
ALTER FUNCTION queue_automatic_pipes() OWNER TO pipes_user;
ALTER FUNCTION queue_pipe_as_first(workspace_id_param INTEGER, key_param VARCHAR(50)) OWNER TO pipes_user;
ALTER FUNCTION remove_locked_from_queue(age INTERVAL) OWNER TO pipes_user;
ALTER FUNCTION remove_synced_from_queue(age INTERVAL) OWNER TO pipes_user;
