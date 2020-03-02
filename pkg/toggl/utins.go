package toggl

import (
	"encoding/json"
)

const (
	maxPayloadSizeBytes = 800 * 1000
)

func AdjustRequestSize(tasks []*Task, split int) ([]*TaskRequest, error) {
	var trs []*TaskRequest
	var size int
	size = len(tasks) / split
	for i := 0; i < split; i++ {
		startIndex := i * size
		endIndex := (i + 1) * size
		if i == split-1 {
			endIndex = len(tasks)
		}
		if endIndex > startIndex {
			t := TaskRequest{
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
			return AdjustRequestSize(tasks, split+1)
		}
	}
	return trs, nil
}
