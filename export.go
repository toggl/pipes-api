package main

import (
	"strconv"
	"time"
)

func fetchTimeEntries(p *Pipe) error {
	return nil
}

func postTimeEntries(p *Pipe) error {
	var err error
	var entriesCon *Connection
	var usersCon, tasksCon, projectsCon *ReversedConnection
	service, err := p.Service()
	if err != nil {
		return err
	}
	if usersCon, err = loadConnectionRev(service, "users"); err != nil {
		return err
	}
	if tasksCon, err = loadConnectionRev(service, "tasks"); err != nil {
		return err
	}
	if projectsCon, err = loadConnectionRev(service, "projects"); err != nil {
		return err
	}
	if entriesCon, err = loadConnection(service, "time_entries"); err != nil {
		return err
	}

	if p.lastSync == nil {
		currentTime := time.Now()
		p.lastSync = &currentTime
	}

	timeEntries, err := getTogglTimeEntries(
		p.authorization.WorkspaceToken, *p.lastSync,
		usersCon.getKeys(), projectsCon.getKeys(),
	)
	if err != nil {
		return err
	}

	for _, entry := range timeEntries {
		entry.foreignID = entriesCon.Data[strconv.Itoa(entry.ID)]
		entry.foreignTaskID = tasksCon.getInt(entry.TaskID)
		entry.foreignUserID = usersCon.getInt(entry.UserID)
		entry.foreignProjectID = projectsCon.getInt(entry.ProjectID)

		entryID, err := service.ExportTimeEntry(&entry)
		if err != nil {
			p.PipeStatus.addError(err)
		} else {
			entriesCon.Data[strconv.Itoa(entry.ID)] = entryID
		}
	}
	if err := entriesCon.save(); err != nil {
		return err
	}
	p.PipeStatus.complete("timeentries", []string{}, len(timeEntries))
	return nil
}
