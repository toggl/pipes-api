package client

import (
	"encoding/json"

	"github.com/toggl/pipes-api/pkg/toggl"
)

const (
	maxPayloadSizeBytes = 800 * 1000
)

func (c *TogglApiClient) AdjustRequestSize(tasks []*toggl.Task, split int) ([]*toggl.TaskRequest, error) {
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
			return c.AdjustRequestSize(tasks, split+1)
		}
	}
	return trs, nil
}
