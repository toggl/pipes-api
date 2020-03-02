package autosync

import (
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/bugsnag/bugsnag-go"

	"github.com/toggl/pipes-api/pkg/integrations"
)

type PipesQueue interface {
	QueueAutomaticPipes() error
	GetPipesFromQueue() ([]*integrations.Pipe, error)
	SetQueuedPipeSynced(*integrations.Pipe) error
	Run(*integrations.Pipe)
}

var wg sync.WaitGroup

const (
	workersCount = 15
	sleepMin     = 60
	sleepMax     = 300
)

type Service struct {
	pipesQueue PipesQueue
}

func NewService(p PipesQueue) *Service {
	return &Service{
		pipesQueue: p,
	}
}

func (ss *Service) Start() {
	go ss.startRunner()
	go ss.startQueue()
}

// run background workers
func (ss *Service) runPipes() {
	wg.Add(workersCount)
	for i := 0; i < workersCount; i++ {
		go ss.pipeWorker(i)
	}
}

// background worker function
func (ss *Service) pipeWorker(id int) {
	defer func() {
		log.Printf("[Workder %d] died\n", id)
		wg.Done()
	}()
	for {
		pipes, err := ss.pipesQueue.GetPipesFromQueue()
		if err != nil {
			bugsnag.Notify(err)
			continue
		}

		// no more work, sleep then continue
		if pipes == nil {
			duration := time.Duration(30+rand.Int31n(30)) * time.Second

			log.Printf("[Worker %d] did not receive works, sleeping for %d\n", id, duration)
			time.Sleep(duration)

			continue
		}

		log.Printf("[Worker %d] received %d pipes\n", id, len(pipes))
		for _, pipe := range pipes {
			log.Printf("[Worker %d] working on pipe [workspace_id: %d, key: %s] starting\n", id, pipe.WorkspaceID, pipe.Key)
			ss.pipesQueue.Run(pipe)

			err := ss.pipesQueue.SetQueuedPipeSynced(pipe)
			if err != nil {
				bugsnag.Notify(err, bugsnag.MetaData{
					"pipe": {
						"ID":            pipe.ID,
						"Name":          pipe.Name,
						"ServiceParams": string(pipe.ServiceParams),
						"WorkspaceID":   pipe.WorkspaceID,
						"ServiceID":     pipe.ServiceID,
					},
				})
			}
			log.Printf("[Worker %d] working on pipe [workspace_id: %d, key: %s] done, err: %t\n", id, pipe.WorkspaceID, pipe.Key, err != nil)
		}
	}
}

func (ss *Service) startRunner() {
	for {
		duration := time.Duration(rand.Intn(sleepMax-sleepMin)+sleepMin) * time.Second
		log.Println("-- Autosync sleeping for ", duration)
		time.Sleep(duration)

		log.Println("-- Autosync started")
		ss.runPipes()

		wg.Wait()
		log.Println("-- Autosync finished")
	}
}

// schedule background job for each integration with auto sync enabled
func (ss *Service) startQueue() {
	for {
		// making sleep longer to not trigger auto sync too fast
		// between 600s until 3000s
		duration := time.Duration(rand.Intn(sleepMax-sleepMin)+sleepMin) * time.Second * 10
		log.Println("-- startQueue sleeping for ", duration)
		time.Sleep(duration)

		log.Println("-- startQueue started")

		if err := ss.pipesQueue.QueueAutomaticPipes(); err != nil {
			if !strings.Contains(err.Error(), `duplicate key value violates unique constraint`) {
				bugsnag.Notify(err)
			}
		}
		log.Println("-- startQueue finished")
	}
}
