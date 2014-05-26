package freshbooks

import (
	"encoding/json"
	"github.com/tambet/oauthplain"
	"io/ioutil"
	"testing"
)

type authConfig struct {
	AccountName      string
	AuthToken        string // Token-Based authentication (deprecated)
	ConsumerKey      string // OAuth authentication
	ConsumerSecret   string // OAuth authentication
	OAuthToken       string // OAuth authentication
	OAuthTokenSecret string // OAuth authentication
}

func loadTestConfig(t *testing.T) *authConfig {
	file, e := ioutil.ReadFile("test-config.json")
	if e != nil {
		t.Fatal("Unable to load 'test-config.json'")
	}
	var config authConfig
	if err := json.Unmarshal(file, &config); err != nil {
		t.Fatal("Unable to unmarshal 'test-config.json'")
	}
	return &config
}

func TestGetUsers(t *testing.T) {
	conf := loadTestConfig(t)
	api := NewApi(conf.AccountName, conf.AuthToken)
	users, err := api.Users()
	if err != nil {
		t.Fatal("Freshbooks retured an error:", err.Error())
	}
	if len(users) < 1 {
		t.Fatal("There should be at least one user")
	}
}

func TestOAuth(t *testing.T) {
	conf := loadTestConfig(t)
	token := &oauthplain.Token{
		ConsumerKey:      conf.ConsumerKey,
		ConsumerSecret:   conf.ConsumerSecret,
		OAuthToken:       conf.OAuthToken,
		OAuthTokenSecret: conf.OAuthTokenSecret,
	}
	api := NewApi(conf.AccountName, token)
	users, err := api.Users()
	if err != nil {
		t.Fatal("Freshbooks retured an error:", err.Error())
	}
	if len(users) < 1 {
		t.Fatal("There should be at least one user")
	}
}
