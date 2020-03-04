package config

import (
	"os"

	"github.com/namsral/flag"
)

const (
	EnvTypeProduction  = "production"
	EnvTypeStaging     = "staging"
	EnvTypeDevelopment = "development"
)

type Flags struct {
	Port          int
	WorkDir       string
	BugsnagAPIKey string
	Environment   string
	DbConnString  string
}

func ParseFlags(flags *Flags) {
	fs := flag.NewFlagSetWithEnvPrefix(os.Args[0], "PIPES_API", flag.ExitOnError)

	fs.IntVar(&flags.Port, "port", 8100, "port")
	fs.StringVar(&flags.WorkDir, "workdir", ".", "Workdir of server")
	fs.StringVar(&flags.BugsnagAPIKey, "bugsnag_key", "", "Bugsnag API Key")
	fs.StringVar(&flags.Environment, "EnvType", EnvTypeDevelopment, "env")
	fs.StringVar(&flags.DbConnString, "db_conn_string", "dbname=pipes_development user=pipes_user host=localhost sslmode=disable port=5432", "DB Connection String")

	fs.Parse(os.Args[1:])
}