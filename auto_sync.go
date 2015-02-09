package main

import (
	"github.com/bugsnag/bugsnag-go"
	"log"
	"sync"
	"time"
)

var wg sync.WaitGroup

const channelCount = 3

func runPipes(chs []chan *Pipe) {
	wg.Add(len(chs))
	for _, ch := range chs {
		go func(ch chan *Pipe) {
			for pipe := range ch {
				pipe.run()
			}
			wg.Done()
		}(ch)
	}
}

func getPipes(chs []chan *Pipe) {
	pipes, err := loadAutomaticPipes()
	if err != nil {
		bugsnag.Notify(err)
	}
	channelSelect := 0
	for _, p := range pipes {
		chs[channelSelect%len(chs)] <- p
		channelSelect++
	}
	for _, ch := range chs {
		close(ch)
	}
	wg.Wait()
}

func makeChannels(n int) []chan *Pipe {
	chs := make([]chan *Pipe, n)
	for i := 0; i < len(chs); i++ {
		chs[i] = make(chan *Pipe)
	}
	return chs
}

func autoSyncRunner() {
	// Sleep 1 minute before starting
	time.Sleep(time.Minute)
	for {
		log.Println("-- Autosync started")
		chs := makeChannels(channelCount)
		runPipes(chs)
		getPipes(chs)
		log.Println("-- Autosync finished")
		time.Sleep(10 * time.Minute)
	}
}
