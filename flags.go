package main

import "flag"

var (
	port          = flag.Int("port", 8100, "port")
	workdir       = flag.String("workdir", ".", "Workdir of server")
	bugsnagAPIKey = flag.String("bugsnag_key", "", "Bugsnag API key")
	environment   = flag.String("environment", "development", "Environment")

	// Database flags
	dbConnString = flag.String("db_conn_string", "dbname=pipes_development user=pipes_user host=localhost sslmode=disable port=5432", "DB Connection String")
)
