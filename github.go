package main

import (
	"code.google.com/p/goauth2/oauth"
	"encoding/json"
	"fmt"
	"github.com/google/go-github/github"
	"strconv"
)

type GithubService struct {
	emptyService
	workspaceID int
	token       oauth.Token
}

func (s *GithubService) Name() string {
	return "github"
}

func (s *GithubService) WorkspaceID() int {
	return s.workspaceID
}

func (s *GithubService) keyFor(objectType string) string {
	return fmt.Sprintf("github:%s", objectType)
}

func (s *GithubService) setAuthData(b []byte) error {
	if err := json.Unmarshal(b, &s.token); err != nil {
		return err
	}
	return nil
}

func (s *GithubService) Accounts() ([]*Account, error) {
	var accounts []*Account
	account := Account{ID: 1, Name: "Self"}
	accounts = append(accounts, &account)
	return accounts, nil
}

// Map Github repos to projects
func (s *GithubService) Projects() ([]*Project, error) {
	repos, _, err := s.client().Repositories.List("", nil)
	if err != nil {
		return nil, err
	}
	var projects []*Project
	for _, object := range repos {
		project := Project{
			Active:    true,
			Name:      *object.Name,
			ForeignID: strconv.Itoa(*object.ID),
		}
		projects = append(projects, &project)
	}
	return projects, nil
}

func (s *GithubService) client() *github.Client {
	t := &oauth.Transport{Token: &s.token}
	return github.NewClient(t.Client())
}
