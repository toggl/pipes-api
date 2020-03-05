package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseFlags(t *testing.T) {
	f := &Flags{}
	args := []string{
		"test",
		"-port",
		"9999",
		"-workdir",
		"./test",
		"-bugsnag_key",
		"TEST_BUGSNAP_KEY",
		"-EnvType",
		"development",
		"-db_conn_string",
		"dbname=pipes host=localhost port=5432",
	}
	ParseFlags(f, args)

	assert.Equal(t, 9999, f.Port)
	assert.Equal(t, "./test", f.WorkDir)
	assert.Equal(t, "TEST_BUGSNAP_KEY", f.BugsnagAPIKey)
	assert.Equal(t, "development", f.Environment)
	assert.Equal(t, "dbname=pipes host=localhost port=5432", f.DbConnString)
}

func TestParseFlagsDefaults(t *testing.T) {
	f := &Flags{}
	ParseFlags(f, make([]string, 1))

	assert.Equal(t, 8100, f.Port)
	assert.Equal(t, ".", f.WorkDir)
	assert.Equal(t, "", f.BugsnagAPIKey)
	assert.Equal(t, EnvTypeDevelopment, f.Environment)
	assert.Equal(t, "dbname=pipes_development user=pipes_user host=localhost sslmode=disable port=5432", f.DbConnString)
}

func TestParseFlagsPanics(t *testing.T) {
	f := &Flags{}
	got := func() { ParseFlags(f, make([]string, 0)) }
	assert.Panics(t, got)
}
