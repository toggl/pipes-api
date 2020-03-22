CREATE ROLE pipes_user WITH LOGIN;

/*
    Table "authorizations" stores authorization tokens (oAuth1/oAuth2) for external services for a given workspace and Toggl token.
    Example of "authorizations" data:
        workspace_id: 1
        workspace_token: test_user
        service: asana
        data: {"AccessToken":"eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJhdXRob3JpemF0aW9uIjoxMTY3NjAzNzAyNTM3NDAwLCJzY29wZSI6ImRlZmF1bHQgaWRlbnRpdHkiLCJzdWIiOjExNjM1MTE4MDEyNjc4OTMsImlhdCI6MTU4NDczOTc2OCwiZXhwIjoxNTg0NzQzMzY4fQ.XUQpxilt9yamxDI-nzWU-tNoWJaJztHrmIQgf360wVM","RefreshToken":"1/1163511801267893:e38d47cd6fb7acdacaf66d3e9da865eb","Expiry":"2020-03-21T01:29:28.172463+03:00","Extra":null}
 */
CREATE TABLE authorizations(
  workspace_id INTEGER,
  workspace_token VARCHAR(50),
  service VARCHAR(50),
  data JSON
);
COMMENT ON COLUMN authorizations.data IS 'This field store 2 types of structures. For OAuth v1 it stores "oauthplain.Token" from "github.com/tambet/oauthplain" library. For OAuth v2 it stores "oauth.Token" from "code.google.com/p/goauth2/oauth" library.';

/*
    Table "imports" stores different type of imported items from external service for each workspace_id and pipe.
    Some examples of "imports" data:
        workspace_id: 1
        key: asana:account:9370405203405:projects
        data: {"error":"","projects":[{"name":"test project","active":true,"foreign_id":"1167139632192241"},{"name":"test 2 project","active":true,"foreign_id":"1167621479402017"}]}
        created_at: 2020-03-20 21:47:10.329287

        workspace_id: 1
        key: asana:account:9370405203405:tasks
        data: {"error":"","tasks":[{"name":"One task","active":true,"pid":1,"foreign_id":"1167621479402021"},{"name":"Another task","active":true,"pid":1,"foreign_id":"1167621479402023"},{"name":"","active":true,"pid":1,"foreign_id":"1167621479402025"},{"name":"Thidr task","active":true,"pid":2,"foreign_id":"1167621479402027"},{"name":"Fourth task","active":true,"pid":2,"foreign_id":"1167621479402029"},{"name":"Six task","active":true,"pid":2,"foreign_id":"1167621479402031"},{"name":"","active":true,"pid":2,"foreign_id":"1167621479402033"}]}
        created_at: 2020-03-20 22:01:04.793722

        workspace_id: 1
        key: asana:account:9370405203405:users
        data: {"error":"","users":[{"email":"support@toggl.com","name":"Toggl • Support","foreign_id":"96440724390141"},{"email":"test.user@toggl.com","name":"Test User","foreign_id":"1163511801267893"}]}
        created_at: 2020-03-20 21:31:52.650098

 */
CREATE TABLE imports(
  workspace_id INTEGER,
  key VARCHAR(50),
  data JSON,
  created_at TIMESTAMP
);

-- CREATE INDEX workspace_imports ON imports USING btree (workspace_id, key);
DROP INDEX CONCURRENTLY IF EXISTS workspace_imports;
CREATE INDEX CONCURRENTLY workspace_imports_at ON imports USING btree (workspace_id, key, created_at);

/*
    Table "pipes" stores configured pipes for each service and specified workspace.
    Example of "pipes" data:
        workspace_id: 1
        key: asana:projects
        data: {"id":"projects","name":"","automatic_option":false,"configured":true,"premium":false,"service_params":"eyJhY2NvdW50X2lkIjo5MzcwNDA1MjAzNDA1fQ=="}
 */
CREATE TABLE pipes(
  workspace_id INTEGER,
  key VARCHAR(50),
  data JSON
);
ALTER TABLE pipes ADD CONSTRAINT pipes_pk PRIMARY KEY (workspace_id, key);

/*
    Table "pipes_status" stores last pipe synchronization status for each service and specified workspace.
    Example of "pipes_status" data:
        workspace_id: 1
        key: asana:tasks
        data: {"status":"success","message":"2 projects, 7 tasks successfully imported/exported","sync_log":"http://0.0.0.0:8100/api/v1/integrations/asana/pipes/tasks/log","sync_date":"2020-03-21T01:00:59+03:00","object_counts":["2 projects","7 tasks"],"notifications":["t1","t2"]}
 */
CREATE TABLE pipes_status(
  workspace_id INTEGER,
  key VARCHAR(50),
  data JSON
);

/*
    Table "connections" stores ID Mappings between External Service items to Toggl items.
    Example of "connections" data:
        workspace_id: 1
        key: asana:account:9370405203405:tasks
        data: {"WorkspaceID":1,"Key":"asana:account:9370405203405:tasks","Data":{"1167621479402021":100,"1167621479402023":200,"1167621479402025":300,"1167621479402027":400,"1167621479402029":500,"1167621479402031":600,"1167621479402033":700}}
 */
CREATE TABLE connections(
  workspace_id INTEGER,
  key VARCHAR(50),
  data JSON
);
COMMENT ON TABLE connections IS 'This table stores ID Mappings for imported items. It maps External Service IDs to Toggl IDs.';

/*
    Table "queued_pipes" is a simple queue for pipes synchronization implemented with PostgreSQL.
    Example of "queued_pipes" data:
        workspace_id: 1
        key: asana:projects
        priority: <null>
        created_at: 2020-03-20 21:46:56.621304
        locked_at: 2020-03-20 21:47:09.249695
        synced_at: 2020-03-20 21:47:10.333947
 */
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
  WITH pending_queue AS (
    SELECT DISTINCT ON (t.workspace_id) t.workspace_id, t.key, t.priority, t.created_at
    FROM (
      SELECT queued_pipes.workspace_id, queued_pipes.key, queued_pipes.priority, queued_pipes.created_at
      FROM queued_pipes
      WHERE locked_at IS NULL AND synced_at IS NULL
      FOR UPDATE
    ) as t
  )
  UPDATE
    queued_pipes
  SET
    locked_at = NOW()
  FROM (
    SELECT pending_queue.workspace_id, pending_queue.key
    FROM pending_queue
    ORDER BY pending_queue.priority DESC, pending_queue.created_at ASC
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
  SELECT r.workspace_id, r.key
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

CREATE ROLE toggl_alerts_user;
ALTER ROLE toggl_alerts_user WITH NOSUPERUSER INHERIT NOCREATEROLE NOCREATEDB LOGIN NOREPLICATION CONNECTION LIMIT 10 PASSWORD 'md55a60e58de3bb5c79bcd17e441b45fd37' VALID UNTIL 'infinity';
GRANT SELECT, UPDATE ON TABLE pipes TO toggl_alerts_user;
