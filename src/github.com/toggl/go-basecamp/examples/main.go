package main

import (
	"../"
	"log"
	"os"
)

func main() {
	var (
		err         error
		accountId   int
		accessToken string
		accounts    []*basecamp.Account
		projects    []*basecamp.Project
		people      []*basecamp.Person
		todoLists   []*basecamp.TodoList
	)

	if accessToken = os.Getenv("BASECAMP_ACCESS_TOKEN"); accessToken == "" {
		log.Println("ERROR: Unable to retrieve BASECAMP_ACCESS_TOKEN environment variable!")
		return
	}

	log.Println("Fetching accounts, projects and users...")
	c := basecamp.Client{AccessToken: accessToken}
	if accounts, err = c.GetAccounts(); err != nil {
		log.Printf("ERROR %q", err)
		return
	}
	accountId = accounts[1].Id

	if projects, err = c.GetProjects(accountId); err != nil {
		log.Printf("ERROR %q", err)
		return
	}

	if people, err = c.GetPeople(accountId); err != nil {
		log.Printf("ERROR %q", err)
		return
	}

	if todoLists, err = c.GetTodoLists(accountId); err != nil {
		log.Printf("ERROR %q", err)
		return
	}

	for _, account := range accounts {
		log.Printf("Account: %+v", account)
	}

	for _, person := range people {
		log.Printf("Person: %+v", person)
	}

	for _, project := range projects {
		log.Printf("Project: %+v", *project)
	}

	for _, todoList := range todoLists {
		log.Printf("Todolist: %+v", *todoList)
	}
}
