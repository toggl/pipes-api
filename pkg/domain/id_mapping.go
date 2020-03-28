package domain

import (
	"strconv"
	"strings"
)

// IDMapping describes service ID connection between external services end Toggl.
//
// It can store "users", "clients", "projects", "tasks", "todolists" or "time_entries" id mappings.
// The "Key" field will store different types of data described above.
// In a different cases "Data" map will store different types, such as:
//
// map[ForeignTaskID]TaskID			(E.g. map["1"] = 112)
// map[ForeignClientID]ClientID		(E.g. map["2"] = 234)
// map[ForeignProjectID]ProjectID	(E.g. map["3"] = 350)
type IDMapping struct {
	WorkspaceID int
	Key         string
	Data        map[string]int
}

func NewIDMapping(workspaceID int, key string) *IDMapping {
	return &IDMapping{
		WorkspaceID: workspaceID,
		Key:         key,
		Data:        make(map[string]int),
	}
}

// ReversedIDMapping describes reversed service id mappings.
//
// It can store "tasks", "users" or "projects" item id mappings.
// In a different cases "Data" map will store different types, such as:
//
// map[TaskID]ForeignTaskID-tasks			(E.g. map[1] = "5-tasks")
// map[UserID]ForeignUserID-users 			(E.g. map[2] = "3-users")
// map[ProjectID]ForeignProjectID-projects	(E.g. map[5] = "8-projects")
type ReversedIDMapping struct {
	Data map[int]string
}

func NewReversedConnection() *ReversedIDMapping {
	return &ReversedIDMapping{map[int]string{}}
}

func (c *ReversedIDMapping) GetForeignID(key int) int {
	res, _ := strconv.Atoi(strings.Split(c.Data[key], "-")[0])
	return res
}

func (c *ReversedIDMapping) GetKeys() []int {
	keys := make([]int, 0, len(c.Data))
	for key := range c.Data {
		keys = append(keys, key)
	}
	return keys
}
