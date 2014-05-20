package freshbooks

import (
	"os"
	"testing"
)

func getCredentials(t *testing.T) (string, string) {
	var account, token string
	if account = os.Getenv("FRESHBOOKS_ACCOUNT"); account == "" {
		t.Fatal("Unable to get FRESHBOOKS_ACCOUNT environment variable")
	}

	if token = os.Getenv("FRESHBOOKS_TOKEN"); token == "" {
		t.Fatal("Unable to get FRESHBOOKS_TOKEN environment variable")
	}
	return account, token
}

func TestGetUsers(t *testing.T) {
	api := NewApi(getCredentials(t))
	users, err := api.Users()

	if err != nil {
		t.Fatal("Freshbooks retured an error:", err.Error())
	}
	if len(users) < 1 {
		t.Fatal("There should be at least one user")
	}
}
