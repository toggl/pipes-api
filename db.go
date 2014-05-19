package main

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"log"
)

var db *sql.DB

func connectDB(host string, port int, name string, user string, pass string) *sql.DB {
	dbConnString := fmt.Sprintf("host=%s port=%d dbname=%s sslmode=disable user=%s password=%s", host, port, name, user, pass)
	result, err := sql.Open("postgres", dbConnString)
	if err != nil {
		log.Fatal(err)
	}
	return result
}
