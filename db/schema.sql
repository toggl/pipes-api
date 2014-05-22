CREATE TABLE authorizations(
  workspace_id INTEGER,
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

ALTER TABLE authorizations OWNER TO pipes_user;
ALTER TABLE imports OWNER TO pipes_user;
ALTER TABLE pipes OWNER TO pipes_user;
ALTER TABLE pipes_status OWNER TO pipes_user;
ALTER TABLE connections OWNER TO pipes_user;
