// +build integration

package main

import (
	"log"
	"os"
	"testing"

	"code.google.com/p/goauth2/oauth"
)

func resetAsanaLimit() {
	asanaPerPageLimit = 100
}

func createAsanaService() Service {
	s := &AsanaService{}
	token := oauth.Token{
		AccessToken: os.Getenv("ASANA_PERSONAL_TOKEN"),
	}
	s.token = token
	s.AsanaParams = &AsanaParams{
		AccountID: numberStrToInt64(os.Getenv("ASANA_ACCOUNT_ID")),
	}
	return s
}

func TestAsanaAccounts(t *testing.T) {
	s := createAsanaService()

	accounts, err := s.Accounts()
	if err != nil {
		t.Error("error calling accounts(), err:", err)
	}

	if len(accounts) != 1 {
		t.Error("should get 1 account returned")
	}
	if accounts[0].ID != numberStrToInt64(os.Getenv("ASANA_ACCOUNT_ID")) {
		t.Error("got wrong account id")
	}
}

func TestAsanaUsers(t *testing.T) {
	s := createAsanaService()

	users, err := s.Users()
	if err != nil {
		t.Error("error calling users(), err:", err)
	}

	if len(users) == 0 {
		t.Error("should get some users")
	}
}

func TestAsanaProjects(t *testing.T) {
	defer resetAsanaLimit()
	asanaPerPageLimit = 10

	s := createAsanaService()

	projects, err := s.Projects()
	if err != nil {
		t.Error("error calling projects(), err:", err)
	}

	if len(projects) <= 10 {
		t.Error("should get more than 10 project, please create at least 10 project to test pagination")
	}
	log.Print(len(projects))
}

func TestAsanaTask(t *testing.T) {
	defer resetAsanaLimit()
	asanaPerPageLimit = 10

	s := createAsanaService()

	tasks, err := s.Tasks()
	if err != nil {
		t.Error("error calling tasks(), err: ", err)
	}

	if len(tasks) <= 10 {
		t.Error(`should get more than 10 tasks, \
please create at least 20 tasks and assign them to a project to test pagination`)
	}
}
