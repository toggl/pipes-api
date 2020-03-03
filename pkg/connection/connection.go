package connection

import (
	"strconv"
	"strings"
)

// Connection describes service connection.
//
// It can store "users", "clients", "projects", "tasks", "todolists" or "time_entries" connections.
// The "Key" field will store different types of data described above.
// In a different cases "Data" map will store different types, such as:
//
// map[ForeignTaskID]TaskID			(E.g. map["1"] = 112)
// map[ForeignClientID]ClientID		(E.g. map["2"] = 234)
// map[ForeignProjectID]ProjectID	(E.g. map["3"] = 350)
type Connection struct {
	WorkspaceID int
	Key         string
	Data        map[string]int
}

func NewConnection(workspaceID int, key string) *Connection {
	return &Connection{
		WorkspaceID: workspaceID,
		Key:         key,
		Data:        make(map[string]int),
	}
}

// ReversedConnection describes reversed service connection.
//
// It can store "tasks", "users" or "projects" connections.
// In a different cases "Data" map will store different types, such as:
//
// map[TaskID]ForeignTaskID-tasks			(E.g. map[1] = "5-tasks")
// map[UserID]ForeignUserID-users 			(E.g. map[2] = "3-users")
// map[ProjectID]ForeignProjectID-projects	(E.g. map[5] = "8-projects")
type ReversedConnection struct {
	Data map[int]string
}

func NewReversedConnection() *ReversedConnection {
	return &ReversedConnection{map[int]string{}}
}

func (c *ReversedConnection) GetForeignID(key int) int {
	res, _ := strconv.Atoi(strings.Split(c.Data[key], "-")[0])
	return res
}

func (c *ReversedConnection) GetKeys() []int {
	keys := make([]int, 0, len(c.Data))
	for key := range c.Data {
		keys = append(keys, key)
	}
	return keys
}
