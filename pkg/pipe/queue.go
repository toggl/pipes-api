package pipe

type QueueRunner interface {
	Queue
	Run(*Pipe)
}

type Queue interface {
	QueueAutomaticPipes() error
	GetPipesFromQueue() ([]*Pipe, error)
	SetQueuedPipeSynced(*Pipe) error
}
