package integration

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"strings"
	"sync"

	"github.com/toggl/pipes-api/pkg/integration"
	"github.com/toggl/pipes-api/pkg/pipe"
)

type FileStorage struct {
	availablePipeType     *regexp.Regexp
	availableServiceType  *regexp.Regexp
	availableIntegrations []*pipe.Integration
	// Stores available authorization types for each service
	// Map format: map[externalServiceID]authType
	availableAuthTypes map[integration.ID]string
	mx                 sync.RWMutex
}

func NewFileStorage(integrationsConfigPath string) *FileStorage {
	svc := &FileStorage{
		availableIntegrations: []*pipe.Integration{},
		availableAuthTypes:    map[integration.ID]string{},
	}
	svc.loadIntegrations(integrationsConfigPath).fillAvailableServices().fillAvailablePipeTypes()
	svc.mx.RLock()
	for _, integration := range svc.availableIntegrations {
		svc.availableAuthTypes[integration.ID] = integration.AuthType
	}
	svc.mx.RUnlock()
	return svc
}

func (fis *FileStorage) IsValidPipe(pipeID integration.PipeID) bool {
	return fis.availablePipeType.MatchString(string(pipeID))
}

func (fis *FileStorage) IsValidService(serviceID integration.ID) bool {
	return fis.availableServiceType.MatchString(string(serviceID))
}

func (fis *FileStorage) LoadAuthorizationType(serviceID integration.ID) (string, error) {
	fis.mx.RLock()
	defer fis.mx.RUnlock()
	return fis.availableAuthTypes[serviceID], nil
}

func (fis *FileStorage) LoadIntegrations() ([]*pipe.Integration, error) {
	fis.mx.RLock()
	defer fis.mx.RUnlock()
	return fis.availableIntegrations, nil
}

func (fis *FileStorage) SaveAuthorizationType(serviceID integration.ID, authType string) error {
	fis.mx.Lock()
	defer fis.mx.Unlock()
	fis.availableAuthTypes[serviceID] = authType
	return nil
}

func (fis *FileStorage) loadIntegrations(integrationsConfigPath string) *FileStorage {
	fis.mx.Lock()
	defer fis.mx.Unlock()
	b, err := ioutil.ReadFile(integrationsConfigPath)
	if err != nil {
		log.Fatalf("Could not read integrations.json, reason: %v", err)
	}
	if err := json.Unmarshal(b, &fis.availableIntegrations); err != nil {
		log.Fatalf("Could not parse integrations.json, reason: %v", err)
	}
	return fis
}

func (fis *FileStorage) fillAvailableServices() *FileStorage {
	ids := make([]string, len(fis.availableIntegrations))
	for i := range fis.availableIntegrations {
		ids = append(ids, string(fis.availableIntegrations[i].ID))
	}
	fis.availableServiceType = regexp.MustCompile(strings.Join(ids, "|"))
	return fis
}

func (fis *FileStorage) fillAvailablePipeTypes() *FileStorage {
	fis.mx.Lock()
	defer fis.mx.Unlock()
	str := fmt.Sprintf("%s|%s|%s|%s|%s|%s", integration.UsersPipe, integration.ProjectsPipe, integration.TodoListsPipe, integration.TodosPipe, integration.TasksPipe, integration.TimeEntriesPipe)
	fis.availablePipeType = regexp.MustCompile(str)
	return fis
}
