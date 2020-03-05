package pipe

//go:generate mockery -name QueueRunner -case underscore -output ./mocks
type QueueRunner interface {
	Queue
	Run(*Pipe)
}

//go:generate mockery -name Queue -case underscore -output ./mocks
type Queue interface {
	QueueAutomaticPipes() error
	GetPipesFromQueue() ([]*Pipe, error)
	SetQueuedPipeSynced(*Pipe) error
}
