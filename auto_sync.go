package main

import (
	"github.com/bugsnag/bugsnag-go"
	"log"
	"time"
)

func autoSyncRunner() {
	// Sleep 1 minute before starting
	time.Sleep(time.Minute)
	for {
		log.Println("-- Autosync started")
		pipes, err := loadAutomaticPipes()
		if err != nil {
			bugsnag.Notify(err)
		}
		for _, pipe := range pipes {
			pipe.run()
		}
		log.Println("-- Autosync finished")
		time.Sleep(10 * time.Minute)
	}
}
