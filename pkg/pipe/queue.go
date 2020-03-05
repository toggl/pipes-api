package pipe

//go:generate mockery -name QueueRunner -case underscore -inpkg
type QueueRunner interface {
	Queue
	Run(*Pipe)
}

//go:generate mockery -name Queue -case underscore -inpkg
type Queue interface {
	QueueAutomaticPipes() error
	GetPipesFromQueue() ([]*Pipe, error)
	SetQueuedPipeSynced(*Pipe) error
	QueuePipeAsFirst(*Pipe) error
}
