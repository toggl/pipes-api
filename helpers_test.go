package main

import (
	"fmt"
	"testing"
)

func generateTasks(nr int) []*Task {
	var ts []*Task
	for i := 0; i < nr; i++ {
		t := Task{ID: i, Name: `Name`, Active: (i%2 == 0), ForeignID: fmt.Sprintf("%d", i), ProjectID: i}
		ts = append(ts, &t)
	}
	return ts
}

func TestTaskSplitting(t *testing.T) {
	taskCount := 100000
	for i := 1; i < 5; i++ {
		ts := generateTasks(taskCount * i)
		tr, err := adjustRequestSize(ts, 1)
		if err != nil {
			t.Error(err)
		}
		if len(tr) != i {
			t.Errorf("Expected split %d\n", i)
		}
	}
}
