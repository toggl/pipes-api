package server

import "github.com/toggl/pipes-api/pkg/integrations"

type ServiceTypeResolver interface {
	AvailableServiceType(serviceID integrations.ExternalServiceID) bool
}

type PipeTypeResolver interface {
	AvailablePipeType(pipeID integrations.PipeID) bool
}
