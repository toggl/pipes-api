package service

import (
	"encoding/json"
	"errors"

	"github.com/toggl/pipes-api/pkg/domain"
	"github.com/toggl/pipes-api/pkg/integration"
)

var ErrNoContent = errors.New("no content")

type PipeService struct {
	pipesStorage domain.PipesStorage
}

func NewPipeService(pipesStorage domain.PipesStorage) *PipeService {
	if pipesStorage == nil {
		panic("PipeService.pipesStorage should not be nil")
	}
	return &PipeService{pipesStorage: pipesStorage}
}

func (svc *PipeService) GetPipe(workspaceID int, serviceID domain.IntegrationID, pipeID domain.PipeID) (*domain.Pipe, error) {
	p := domain.NewPipe(workspaceID, serviceID, pipeID)
	if err := svc.pipesStorage.Load(p); err != nil {
		return nil, err
	}
	var err error
	p.PipeStatus, err = svc.pipesStorage.LoadStatus(workspaceID, serviceID, pipeID)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (svc *PipeService) CreatePipe(workspaceID int, serviceID domain.IntegrationID, pipeID domain.PipeID, params []byte) error {
	p := domain.NewPipe(workspaceID, serviceID, pipeID)

	service := integration.NewPipeIntegration(serviceID, workspaceID)
	err := service.SetParams(params)
	if err != nil {
		return SetParamsError{err}
	}
	p.ServiceParams = params

	if err := svc.pipesStorage.Save(p); err != nil {
		return err
	}
	return nil
}

func (svc *PipeService) UpdatePipe(workspaceID int, serviceID domain.IntegrationID, pipeID domain.PipeID, params []byte) error {
	p := domain.NewPipe(workspaceID, serviceID, pipeID)
	if err := svc.pipesStorage.Load(p); err != nil {
		return err
	}
	if !p.Configured {
		return ErrPipeNotConfigured
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return err
	}
	if err := svc.pipesStorage.Save(p); err != nil {
		return err
	}
	return nil
}

func (svc *PipeService) DeletePipe(workspaceID int, serviceID domain.IntegrationID, pipeID domain.PipeID) error {
	p := domain.NewPipe(workspaceID, serviceID, pipeID)
	if err := svc.pipesStorage.Load(p); err != nil {
		return err
	}
	if err := svc.pipesStorage.Delete(p, workspaceID); err != nil {
		return err
	}
	return nil
}
