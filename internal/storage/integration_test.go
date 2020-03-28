package storage

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/toggl/pipes-api/pkg/domain"
	"github.com/toggl/pipes-api/pkg/integration"
)

const testIntegrations = `
[
	{
		"id": "basecamp",
		"name": "Basecamp",
		"auth_type": "oauth2",
		"image": "logo-basecamp.png",
		"link": "https://localhost/basecamp-integration",
		"pipes": [{
				"id": "users",
				"name": "Users",
				"premium": false,
				"automatic_option": false,
				"description": "Test users pipe description"
			}]
	}
]
`

func TestIntegrationStorage_IsValidPipe(t *testing.T) {

	r := strings.NewReader(testIntegrations)
	is := NewIntegrationStorage(r)

	assert.True(t, is.IsValidPipe(integration.UsersPipe))
}

func TestIntegrationStorage_IsValidService(t *testing.T) {
	r := strings.NewReader(testIntegrations)
	is := NewIntegrationStorage(r)

	assert.True(t, is.IsValidService(integration.BaseCamp))
}

func TestIntegrationStorage_LoadAuthorizationType(t *testing.T) {
	r := strings.NewReader(testIntegrations)
	is := NewIntegrationStorage(r)
	s, err := is.LoadAuthorizationType(integration.BaseCamp)
	assert.NoError(t, err)
	assert.Equal(t, domain.TypeOauth2, s)
}

func TestIntegrationStorage_LoadIntegrations(t *testing.T) {
	r := strings.NewReader(testIntegrations)
	is := NewIntegrationStorage(r)

	igs, err := is.LoadIntegrations()
	assert.NoError(t, err)
	assert.NotNil(t, igs)
	assert.Equal(t, 1, len(igs))
}

func TestIntegrationStorage_SaveAuthorizationType(t *testing.T) {
	r := strings.NewReader(testIntegrations)
	is := NewIntegrationStorage(r)

	err := is.SaveAuthorizationType(integration.BaseCamp, domain.TypeOauth1)
	assert.NoError(t, err)

	s, err := is.LoadAuthorizationType(integration.BaseCamp)
	assert.Equal(t, domain.TypeOauth1, s)
}

func TestNewIntegrationStorage(t *testing.T) {
	r := strings.NewReader(testIntegrations)
	is := NewIntegrationStorage(r)
	assert.NotNil(t, is)
}
