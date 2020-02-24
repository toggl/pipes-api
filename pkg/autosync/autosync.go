package autosync

import (
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/bugsnag/bugsnag-go"

	"github.com/toggl/pipes-api/pkg/pipes"
)

var wg sync.WaitGroup

const (
	workersCount = 15
	sleepMin     = 60
	sleepMax     = 300
)

type Service struct {
	Environment string
	PipeService *pipes.PipeService
}

func (ss *Service) Start() {
	if ss.Environment == "production" {
		go ss.startRunner()
	}
	if ss.Environment == "staging" {
		go ss.startStubRunner()
	}
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
		pipes, err := ss.PipeService.GetPipesFromQueue()
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
			ss.PipeService.Run(pipe)

			err := ss.PipeService.SetQueuedPipeSynced(pipe)
			if err != nil {
				pipe.BugsnagNotifyPipe(err)
			}
			log.Printf("[Worker %d] working on pipe [workspace_id: %d, key: %s] done, err: %t\n", id, pipe.WorkspaceID, pipe.Key, (err != nil))
		}
	}
}

// run dummy background workers
func (ss *Service) runPipesStub() {
	wg.Add(workersCount)
	for i := 0; i < workersCount; i++ {
		go ss.pipeWorkerStub()
	}
}

// dummy background worker function
func (ss *Service) pipeWorkerStub() {
	ranCount := 0
	gotCount := 0
	defer func() {
		log.Printf("Got %d pipes, ran %d pipes\n", gotCount, ranCount)
		wg.Done()
	}()
	for {
		pipes, err := ss.PipeService.GetPipesFromQueue()
		if err != nil {
			bugsnag.Notify(err)
			continue
		}
		if pipes == nil {
			continue
		}
		gotCount += len(pipes)
		for _, pipe := range pipes {
			// NO PIPE RUN HERE
			err := ss.PipeService.SetQueuedPipeSynced(pipe)
			if err != nil {
				log.Printf("ERROR: %s\n", err.Error())
			}
			ranCount++
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

func (ss *Service) startStubRunner() {
	for {
		duration := time.Duration(rand.Intn(sleepMax-sleepMin)+sleepMin) * time.Second
		log.Println("-- AutosyncStub sleeping for ", duration)
		time.Sleep(duration)

		log.Println("-- AutosyncStub started")
		ss.runPipesStub()

		wg.Wait()
		log.Println("-- AutosyncStub finished")
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

		if err := ss.PipeService.QueueAutomaticPipes(); err != nil {
			if !strings.Contains(err.Error(), `duplicate key value violates unique constraint`) {
				bugsnag.Notify(err)
			}
		}
		log.Println("-- startQueue finished")
	}
}
