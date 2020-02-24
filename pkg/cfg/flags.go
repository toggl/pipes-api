package cfg

import (
	"os"

	"github.com/namsral/flag"
)

type Flags struct {
	Port             int
	WorkDir          string
	BugsnagAPIKey    string
	Environment      string
	DbConnString     string
	TestDBConnString string
}

func ParseFlags(flags *Flags) {
	fs := flag.NewFlagSetWithEnvPrefix(os.Args[0], "PIPES_API", flag.ExitOnError)

	fs.IntVar(&flags.Port, "port", 8100, "port")
	fs.StringVar(&flags.WorkDir, "workdir", ".", "Workdir of server")
	fs.StringVar(&flags.BugsnagAPIKey, "bugsnag_key", "", "Bugsnag API Key")
	fs.StringVar(&flags.Environment, "environment", "development", "Environment")
	fs.StringVar(&flags.DbConnString, "db_conn_string", "dbname=pipes_development user=pipes_user host=localhost sslmode=disable port=5432", "DB Connection String")
	fs.StringVar(&flags.TestDBConnString, "test_db_conn_string", "dbname=pipes_test user=pipes_user host=localhost sslmode=disable port=5432", "test DB Connection String")

	fs.Parse(os.Args[1:])
}
