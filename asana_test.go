package main

import (
	"os"
	"testing"

	"code.google.com/p/goauth2/oauth"
)

func createAsanaService() Service {
	s := &AsanaService{}
	token := oauth.Token{
		AccessToken: os.Getenv("ASANA_PERSONAL_TOKEN"),
	}
	s.token = token
	s.AsanaParams = &AsanaParams{
		AccountID: numberStrToInt64("ASANA_ACCOUNT_ID"),
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
	s := createAsanaService()

	projects, err := s.Projects()
	if err != nil {
		t.Error("error calling projects(), err:", err)
	}

	if len(projects) <= 20 {
		t.Error("should get more than 20 project, please create at least 20 project to test pagination")
	}
}

func TestAsanaTask(t *testing.T) {
	s := createAsanaService()

	tasks, err := s.Tasks()
	if err != nil {
		t.Error("error calling tasks(), err: ", err)
	}

	if len(tasks) <= 20 {
		t.Error("should get more than 20 tasks, please create at least 20 tasks to test pagination")
	}
}
