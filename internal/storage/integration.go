package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"regexp"
	"strings"
	"sync"

	"github.com/toggl/pipes-api/pkg/domain"
)

type IntegrationStorage struct {
	availablePipeType     *regexp.Regexp
	availableServiceType  *regexp.Regexp
	availableIntegrations []*domain.Integration
	// Stores available authorization types for each service
	// Map format: map[externalServiceID]authType
	availableAuthTypes map[domain.IntegrationID]string
	mx                 sync.RWMutex
}

func NewIntegrationStorage(configFile io.Reader) *IntegrationStorage {
	svc := &IntegrationStorage{
		availableIntegrations: []*domain.Integration{},
		availableAuthTypes:    map[domain.IntegrationID]string{},
	}
	svc.loadIntegrations(configFile).fillAvailableServices().fillAvailablePipeTypes()
	svc.mx.RLock()
	for _, integration := range svc.availableIntegrations {
		svc.availableAuthTypes[integration.ID] = integration.AuthType
	}
	svc.mx.RUnlock()
	return svc
}

func (is *IntegrationStorage) IsValidPipe(pipeID domain.PipeID) bool {
	return is.availablePipeType.MatchString(string(pipeID))
}

func (is *IntegrationStorage) IsValidService(serviceID domain.IntegrationID) bool {
	return is.availableServiceType.MatchString(string(serviceID))
}

func (is *IntegrationStorage) LoadAuthorizationType(serviceID domain.IntegrationID) (string, error) {
	is.mx.RLock()
	defer is.mx.RUnlock()
	return is.availableAuthTypes[serviceID], nil
}

func (is *IntegrationStorage) LoadIntegrations() ([]*domain.Integration, error) {
	is.mx.RLock()
	defer is.mx.RUnlock()
	return is.availableIntegrations, nil
}

func (is *IntegrationStorage) SaveAuthorizationType(serviceID domain.IntegrationID, authType string) error {
	is.mx.Lock()
	defer is.mx.Unlock()
	is.availableAuthTypes[serviceID] = authType
	return nil
}

func (is *IntegrationStorage) loadIntegrations(configFile io.Reader) *IntegrationStorage {
	is.mx.Lock()
	defer is.mx.Unlock()
	b, err := ioutil.ReadAll(configFile)
	if err != nil {
		log.Fatalf("Could not read integrations.json, reason: %v", err)
	}
	if err := json.Unmarshal(b, &is.availableIntegrations); err != nil {
		log.Fatalf("Could not parse integrations.json, reason: %v", err)
	}
	return is
}

func (is *IntegrationStorage) fillAvailableServices() *IntegrationStorage {
	ids := make([]string, len(is.availableIntegrations))
	for i := range is.availableIntegrations {
		ids = append(ids, string(is.availableIntegrations[i].ID))
	}
	is.availableServiceType = regexp.MustCompile(strings.Join(ids, "|"))
	return is
}

func (is *IntegrationStorage) fillAvailablePipeTypes() *IntegrationStorage {
	is.mx.Lock()
	defer is.mx.Unlock()
	str := fmt.Sprintf("%s|%s|%s|%s|%s|%s", domain.UsersPipe, domain.ProjectsPipe, domain.TodoListsPipe, domain.TodosPipe, domain.TasksPipe, domain.TimeEntriesPipe)
	is.availablePipeType = regexp.MustCompile(str)
	return is
}
