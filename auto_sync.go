package main

import (
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/bugsnag/bugsnag-go"
)

var wg sync.WaitGroup

const (
	workersCount = 15
	sleepMin     = 60
	sleepMax     = 300
)

// run background workers
func runPipes() {
	wg.Add(workersCount)
	for i := 0; i < workersCount; i++ {
		go pipeWorker(i)
	}
}

// background worker function
func pipeWorker(id int) {
	defer func() {
		log.Printf("[Workder %d] died\n", id)
		wg.Done()
	}()
	for {
		pipes, err := getPipesFromQueue()
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
			log.Printf("[Worker %d] working on pipe [workspace_id: %d, key: %s] starting\n", id, pipe.workspaceID, pipe.key)
			pipe.run()

			err := setQueuedPipeSynced(pipe)
			if err != nil {
				BugsnagNotifyPipe(pipe, err)
			}
			log.Printf("[Worker %d] working on pipe [workspace_id: %d, key: %s] done, err: %t\n", id, pipe.workspaceID, pipe.key, (err != nil))
		}
	}
}

// run dummy background workers
func runPipesStub() {
	wg.Add(workersCount)
	for i := 0; i < workersCount; i++ {
		go pipeWorkerStub()
	}
}

// dummy background worker function
func pipeWorkerStub() {
	ranCount := 0
	gotCount := 0
	defer func() {
		log.Printf("Got %d pipes, ran %d pipes\n", gotCount, ranCount)
		wg.Done()
	}()
	for {
		pipes, err := getPipesFromQueue()
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
			err := setQueuedPipeSynced(pipe)
			if err != nil {
				log.Printf("ERROR: %s\n", err.Error())
			}
			ranCount++
		}
	}
}

func autoSyncRunner() {
	for {
		duration := time.Duration(rand.Intn(sleepMax-sleepMin)+sleepMin) * time.Second
		log.Println("-- Autosync sleeping for ", duration)
		time.Sleep(duration)

		log.Println("-- Autosync started")
		runPipes()

		wg.Wait()
		log.Println("-- Autosync finished")
	}
}

func autoSyncRunnerStub() {
	for {
		duration := time.Duration(rand.Intn(sleepMax-sleepMin)+sleepMin) * time.Second
		log.Println("-- AutosyncStub sleeping for ", duration)
		time.Sleep(duration)

		log.Println("-- AutosyncStub started")
		runPipesStub()

		wg.Wait()
		log.Println("-- AutosyncStub finished")
	}
}

// schedule background job for each integration with auto sync enabled
func autoSyncQueuer() {
	for {
		// making sleep longer to not trigger auto sync too fast
		// between 600s until 3000s
		duration := time.Duration(rand.Intn(sleepMax-sleepMin)+sleepMin) * time.Second * 10
		log.Println("-- Queuer sleeping for ", duration)
		time.Sleep(duration)

		log.Println("-- Queuer started")
		_, err := db.Exec(queueAutomaticPipesSQL)
		if err != nil {
			if !strings.Contains(err.Error(), `duplicate key value violates unique constraint`) {
				bugsnag.Notify(err)
			}
		}
		log.Println("-- Queuer finished")
	}
}
