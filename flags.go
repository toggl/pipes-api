package main

import "flag"

var (
	port          = flag.Int("port", 8100, "port")
	workdir       = flag.String("workdir", ".", "Workdir of server")
	bugsnagAPIKey = flag.String("bugsnag_key", "", "Bugsnag API key")
	environment   = flag.String("environment", "development", "Environment")

	// Database flags
	dbUser = flag.String("dbuser", "pipes_user", "DB user")
	dbPass = flag.String("dbpass", "", "DB password")
	dbName = flag.String("dbname", "pipes_development", "DB name")
	dbHost = flag.String("dbhost", "localhost", "DB host")
	dbPort = flag.Int("dbport", 5432, "DB port")
)
