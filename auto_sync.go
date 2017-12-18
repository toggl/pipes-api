package main

import (
	"github.com/bugsnag/bugsnag-go"
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"
)

var wg sync.WaitGroup

const (
	workersCount = 5
	sleepMin     = 300
	sleepMax     = 900
)

func runPipes() {
	wg.Add(workersCount)
	for i := 0; i < workersCount; i++ {
		go pipeWorker()
	}
}

func pipeWorker() {
	for {
		pipes, err := getPipesFromQueue()
		if err != nil {
			bugsnag.Notify(err)
			break
		}
		if pipes == nil {
			break
		}
		for _, pipe := range pipes {
			pipe.run()
			err := setQueuedPipeSynced(pipe)
			if err != nil {
				BugsnagNotifyPipe(pipe, err)
			}
		}
	}
	wg.Done()
}

func runPipesStub() {
	wg.Add(workersCount)
	for i := 0; i < workersCount; i++ {
		go pipeWorkerStub()
	}
}

func pipeWorkerStub() {
	ranCount := 0
	gotCount := 0
	for {
		pipes, err := getPipesFromQueue()
		if err != nil {
			bugsnag.Notify(err)
			break
		}
		if pipes == nil {
			break
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
	log.Printf("Got %d pipes, ran %d pipes\n", gotCount, ranCount)
	wg.Done()
}

func autoSyncRunner() {
	for {
		time.Sleep(time.Duration(rand.Intn(sleepMax-sleepMin)+sleepMin) * time.Second)
		log.Println("-- Autosync started")
		runPipes()
		wg.Wait()
		log.Println("-- Autosync finished")
	}
}

func autoSyncRunnerStub() {
	for {
		time.Sleep(time.Duration(rand.Intn(sleepMax-sleepMin)+sleepMin) * time.Second)
		log.Println("-- AutosyncStub started")
		runPipesStub()
		wg.Wait()
		log.Println("-- AutosyncStub finished")
	}
}

func autoSyncQueuer() {
	for {
		time.Sleep(time.Duration(rand.Intn(sleepMax-sleepMin)+sleepMin) * time.Second)
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
