package connection

import (
	"strconv"
	"strings"
)

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

type ReversedConnection struct {
	Data map[int]string
}

func (c *ReversedConnection) GetInt(key int) int {
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
