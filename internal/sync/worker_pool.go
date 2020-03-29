package sync

import (
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/bugsnag/bugsnag-go"

	"github.com/toggl/pipes-api/pkg/domain"
)

var wg sync.WaitGroup

const (
	workersCount = 15
	sleepMin     = 30
	sleepMax     = 60
)

type WorkerPool struct {
	Debug       bool
	Queue       domain.Queue
	PipeService domain.PipeService
}

func (s *WorkerPool) Start() {
	go s.startRunner()
	go s.startQueue()
}

// background worker function
func (s *WorkerPool) pipeWorker(id int) {
	defer func() {
		s.debugf("[Worker %d] died\n", id)
		wg.Done()
	}()
	for {
		pipes, err := s.Queue.LoadScheduledPipes()
		if err != nil {
			bugsnag.Notify(err)
			continue
		}

		// no more work, sleep then continue
		if pipes == nil {
			duration := time.Duration(30+rand.Int31n(30)) * time.Second
			s.debugf("[Worker %d] did not receive works, sleeping for %d\n", id, duration)
			time.Sleep(duration)

			continue
		}

		log.Printf("[Worker %d] received %d pipes\n", id, len(pipes))
		for _, pipe := range pipes {
			s.debugf("[Worker %d] working on pipe [workspace_id: %d, key: %s] starting\n", id, pipe.WorkspaceID, pipe.Key())

			s.PipeService.Synchronize(pipe)

			err := s.Queue.MarkPipeSynchronized(pipe)
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
			s.debugf("[Worker %d] working on pipe [workspace_id: %d, key: %s] done, err: %t\n", id, pipe.WorkspaceID, pipe.Key(), err != nil)
		}
	}
}

func (s *WorkerPool) startRunner() {
	for {
		duration := time.Duration(rand.Intn(sleepMax-sleepMin)+sleepMin) * time.Second
		s.debugf("-- Autosync sleeping for %s", duration)
		time.Sleep(duration)

		log.Println("-- Autosync started")
		wg.Add(workersCount)
		for i := 0; i < workersCount; i++ {
			go s.pipeWorker(i)
		}
		wg.Wait()
		log.Println("-- Autosync finished")
	}
}

// schedule background job for each integration with auto sync enabled
func (s *WorkerPool) startQueue() {
	for {
		// making sleep longer to not trigger auto sync too fast
		// between 600s until 3000s
		duration := time.Duration(rand.Intn(sleepMax-sleepMin)+sleepMin) * time.Second * 10
		s.debugf("-- startQueue sleeping for %s", duration)
		time.Sleep(duration)

		log.Println("-- startQueue started")

		if err := s.Queue.ScheduleAutomaticPipesSynchronization(); err != nil {
			if !strings.Contains(err.Error(), `duplicate key value violates unique constraint`) {
				bugsnag.Notify(err)
			}
		}
		log.Println("-- startQueue finished")
	}
}

func (s *WorkerPool) debugf(format string, v ...interface{}) {
	if s.Debug {
		log.Printf(format, v...)
	}
}
