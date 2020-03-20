package autosync

import (
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/bugsnag/bugsnag-go"

	"github.com/toggl/pipes-api/pkg/pipe"
)

var wg sync.WaitGroup

const (
	workersCount = 15
	sleepMin     = 60
	sleepMax     = 300
)

type Service struct {
	queue  pipe.Queue
	runner pipe.Runner
}

func NewService(p pipe.Queue, r pipe.Runner) *Service {
	return &Service{
		queue:  p,
		runner: r,
	}
}

func (s *Service) Start() {
	go s.startRunner()
	go s.startQueue()
}

// run background workers
func (s *Service) runPipes() {
	wg.Add(workersCount)
	for i := 0; i < workersCount; i++ {
		go s.pipeWorker(i)
	}
}

// background worker function
func (s *Service) pipeWorker(id int) {
	defer func() {
		log.Printf("[Workder %d] died\n", id)
		wg.Done()
	}()
	for {
		pipes, err := s.queue.GetPipesFromQueue()
		if err != nil {
			bugsnag.Notify(err)
			continue
		}

		// no more work, sleep then continue
		if pipes == nil {
			duration := time.Duration(30+rand.Int31n(30)) * time.Second
			//log.Printf("[Worker %d] did not receive works, sleeping for %d\n", id, duration)
			time.Sleep(duration)

			continue
		}

		log.Printf("[Worker %d] received %d pipes\n", id, len(pipes))
		for _, pipe := range pipes {
			log.Printf("[Worker %d] working on pipe [workspace_id: %d, key: %s] starting\n", id, pipe.WorkspaceID, pipe.Key)
			s.runner.Run(pipe)

			err := s.queue.SetQueuedPipeSynced(pipe)
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

func (s *Service) startRunner() {
	for {
		duration := time.Duration(rand.Intn(sleepMax-sleepMin)+sleepMin) * time.Second
		log.Println("-- Autosync sleeping for ", duration)
		time.Sleep(duration)

		log.Println("-- Autosync started")
		s.runPipes()

		wg.Wait()
		log.Println("-- Autosync finished")
	}
}

// schedule background job for each integration with auto sync enabled
func (s *Service) startQueue() {
	for {
		// making sleep longer to not trigger auto sync too fast
		// between 600s until 3000s
		duration := time.Duration(rand.Intn(sleepMax-sleepMin)+sleepMin) * time.Second * 10
		log.Println("-- startQueue sleeping for ", duration)
		time.Sleep(duration)

		log.Println("-- startQueue started")

		if err := s.queue.QueueAutomaticPipes(); err != nil {
			if !strings.Contains(err.Error(), `duplicate key value violates unique constraint`) {
				bugsnag.Notify(err)
			}
		}
		log.Println("-- startQueue finished")
	}
}
