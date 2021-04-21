package main

import (
	"os"
	"testing"

	"code.google.com/p/goauth2/oauth"
)

func createGithubService() Service {
	s := &GithubService{}
	token := oauth.Token{
		AccessToken: os.Getenv("GITHUB_PERSONAL_TOKEN"),
	}
	s.token = token
	return s
}

func TestGithubProjects(t *testing.T) {
	if os.Getenv("GITHUB_PERSONAL_TOKEN") == "" {
		t.Skipf("missing GITHUB_PERSONAL_TOKEN in env")
	}

	s := createGithubService()

	projects, err := s.Projects()
	if err != nil {
		t.Error("error calling Projects, err:", err)
	}

	if len(projects) == 0 {
		t.Error("should return some projects")
	}
}
