package storage

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
)

type Storage struct {
	ConnString string
	*sql.DB
}

func (sg *Storage) Connect() *sql.DB {
	var err error
	sg.DB, err = sql.Open("postgres", sg.ConnString)
	if err != nil {
		log.Fatal(err)
	}
	return sg.DB
}

func (sg *Storage) IsDown() bool {
	if _, err := sg.DB.Exec("SELECT 1"); err != nil {
		return true
	}
	return false
}
