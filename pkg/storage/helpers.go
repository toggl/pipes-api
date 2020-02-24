package storage

import (
	"encoding/json"
	"strings"

	"github.com/toggl/pipes-api/pkg/toggl"
)

func adjustRequestSize(tasks []*toggl.Task, split int) ([]*toggl.TaskRequest, error) {
	var trs []*toggl.TaskRequest
	var size int
	size = len(tasks) / split
	for i := 0; i < split; i++ {
		startIndex := i * size
		endIndex := (i + 1) * size
		if i == split-1 {
			endIndex = len(tasks)
		}
		if endIndex > startIndex {
			t := toggl.TaskRequest{
				Tasks: tasks[startIndex:endIndex],
			}
			trs = append(trs, &t)
		}
	}
	for _, tr := range trs {
		j, err := json.Marshal(tr)
		if err != nil {
			return nil, err
		}
		if len(j) > maxPayloadSizeBytes {
			return adjustRequestSize(tasks, split+1)
		}
	}
	return trs, nil
}

func trimSpacesFromName(ps []*toggl.Project) []*toggl.Project {
	var trimmedPs []*toggl.Project
	for _, p := range ps {
		p.Name = strings.TrimSpace(p.Name)
		if len(p.Name) > 0 {
			trimmedPs = append(trimmedPs, p)
		}
	}
	return trimmedPs
}
