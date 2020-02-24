package github

import (
	"os"
	"testing"

	"code.google.com/p/goauth2/oauth"
)

func TestGithubProjects(t *testing.T) {
	s := &Service{}
	token := oauth.Token{
		AccessToken: os.Getenv("GITHUB_PERSONAL_TOKEN"),
	}
	s.token = token

	projects, err := s.Projects()
	if err != nil {
		t.Error("error calling Projects, err:", err)
	}

	if len(projects) == 0 {
		t.Error("should return some projects")
	}
}
