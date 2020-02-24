package server

type ServiceTypeResolver interface {
	AvailableServiceType(serviceID string) bool
}

type PipeTypeResolver interface {
	AvailablePipeType(pipeID string) bool
}
