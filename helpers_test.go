package main

import (
	"encoding/json"
	"fmt"
	"strings"
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
	taskCount := 9007
	for i := 1; i < 5; i++ {
		ts := generateTasks(taskCount * i)
		trs, err := adjustRequestSize(ts, 1)
		if err != nil {
			t.Error(err)
		}
		if len(trs) != i {
			t.Errorf("Expected split %d\n", i)
		}
		recievedTaskCount := 0
		for _, tr := range trs {
			recievedTaskCount += len(tr.Tasks)
		}
		if recievedTaskCount != taskCount*i {
			t.Errorf("Expected to get %d tasks but got %d", taskCount, recievedTaskCount)
		}
	}
}

func TestTaskSplittingSmallCount(t *testing.T) {
	ts := generateTasks(3)
	trs, err := adjustRequestSize(ts, 3)
	if err != nil {
		t.Error(err)
	}
	if len(trs) != 3 {
		t.Error("Expected split 3")
	}
	for _, tr := range trs {
		if len(tr.Tasks) != 1 {
			t.Error("Expected 1 task per request")
		}
	}
}

func TestTaskSplittingSmallDifferent(t *testing.T) {
	counts := []int{3, 5, 7, 11, 13, 17, 19, 23, 29, 31, 37, 41, 43, 47, 53, 59, 61, 67, 71, 73, 79, 83, 89, 97, 101, 103, 107, 109, 113, 127, 131, 137, 139, 149, 151, 157, 163, 167, 173, 179, 181, 191, 193, 197, 199, 211, 223, 227, 229, 233, 239, 241, 251, 257, 263, 269, 271, 277, 281, 283, 293, 307, 311, 313, 317, 331, 337, 347, 349, 353, 359, 367, 373, 379, 383, 389, 397, 401, 409, 419, 421, 431, 433, 439, 443, 449, 457, 461, 463, 467, 479, 487, 491, 499, 503, 509, 521, 523, 541, 547, 557, 563, 569, 571, 577, 587, 593, 599, 601, 607, 613, 617, 619, 631, 641, 643, 647, 653, 659, 661, 673, 677, 683, 691, 701, 709, 719, 727, 733, 739, 743, 751, 757, 761, 769, 773, 787, 797, 809, 811, 821, 823, 827, 829, 839, 853, 857, 859, 863, 877, 881, 883, 887, 907, 911, 919, 929, 937, 941, 947, 953, 967, 971, 977, 983, 991, 997}
	for _, c := range counts {
		ts := generateTasks(c)
		trs, err := adjustRequestSize(ts, 3)
		if err != nil {
			t.Error(err)
		}
		if len(trs) != 3 {
			t.Error("Expected split 3")
		}
		totalCount := 0
		for _, tr := range trs {
			totalCount += len(tr.Tasks)
		}
		if totalCount != c {
			t.Errorf("Expected total of %d tasks but got %d\n", c, totalCount)
		}
	}
}

func TestGetProjects(t *testing.T) {
	db = connectDB(*dbHost, *dbPort, testDB, *dbUser, *dbPass)

	p := NewPipe(1, TestServiceName, "projects")

	fetchProjects(p)

	b, err := getObject(p.Service(), "projects")
	if err != nil {
		t.Error(err)
	}
	var pr ProjectsResponse
	err = json.Unmarshal(b, &pr)
	if err != nil {
		t.Error(err)
	}
	if len(pr.Projects) != 4 {
		t.Errorf("Expected 4 projects but got %d", len(pr.Projects))
	}
	if pr.Projects[0].Name != strings.TrimSpace(p1Name) {
		t.Errorf("Expected name '%s' but got '%s'", strings.TrimSpace(p1Name), pr.Projects[0].Name)
	}
	if pr.Projects[1].Name != strings.TrimSpace(p2Name) {
		t.Errorf("Expected name '%s' but got '%s'", strings.TrimSpace(p2Name), pr.Projects[1].Name)
	}
	if pr.Projects[2].Name != strings.TrimSpace(p3Name) {
		t.Errorf("Expected name '%s' but got '%s'", strings.TrimSpace(p3Name), pr.Projects[2].Name)
	}
	if pr.Projects[3].Name != strings.TrimSpace(p4Name) {
		t.Errorf("Expected name '%s' but got '%s'", strings.TrimSpace(p4Name), pr.Projects[3].Name)
	}
}
