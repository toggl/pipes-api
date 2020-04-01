package service

import (
	"errors"

	"github.com/toggl/pipes-api/pkg/domain"
)

type HealthCheckService struct {
	pipesStorage domain.PipesStorage
	togglClient  domain.TogglClient
}

func NewHealthCheckService(pipesStorage domain.PipesStorage, togglClient domain.TogglClient) *HealthCheckService {
	if pipesStorage == nil {
		panic("HealthCheckService.pipesStorage should not be nil")
	}
	if togglClient == nil {
		panic("HealthCheckService.togglClient should not be nil")
	}
	return &HealthCheckService{pipesStorage: pipesStorage, togglClient: togglClient}
}

func (svc *HealthCheckService) Ready() []error {
	errs := make([]error, 0)

	if svc.pipesStorage.IsDown() {
		errs = append(errs, errors.New("database is down"))
	}

	if err := svc.togglClient.Ping(); err != nil {
		errs = append(errs, err)
	}
	return errs
}
