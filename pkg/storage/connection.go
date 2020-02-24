package storage

import (
	"strconv"
	"strings"

	"github.com/toggl/pipes-api/pkg/integrations"
)

type Connection struct {
	workspaceID int
	serviceID   string
	pipeID      string
	key         string
	Data        map[string]int
}

func NewConnection(s integrations.Integration, pipeID string) *Connection {
	return &Connection{
		workspaceID: s.GetWorkspaceID(),
		key:         s.KeyFor(pipeID),
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
