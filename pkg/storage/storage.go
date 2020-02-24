package storage

import (
	_ "github.com/lib/pq"
)

const maxPayloadSizeBytes = 800 * 1000

const (
	usersPipeID    = "users"
	clientsPipeID  = "clients"
	projectsPipeID = "projects"
	tasksPipeId    = "tasks"
	todoPipeId     = "todolists"
)
