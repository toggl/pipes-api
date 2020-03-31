package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"code.google.com/p/goauth2/oauth"
	"github.com/google/go-github/github"

	"github.com/toggl/pipes-api/pkg/domain"
)

type GitHubPipeIntegration struct {
	WorkspaceID int
	token       oauth.Token
}

func (s *GitHubPipeIntegration) ID() domain.IntegrationID {
	return domain.GitHub
}

func (s *GitHubPipeIntegration) GetWorkspaceID() int {
	return s.WorkspaceID
}

func (s *GitHubPipeIntegration) KeyFor(objectType domain.PipeID) string {
	return fmt.Sprintf("github:%s", objectType)
}

func (s *GitHubPipeIntegration) SetAuthData(b []byte) error {
	if err := json.Unmarshal(b, &s.token); err != nil {
		return err
	}
	return nil
}

func (s *GitHubPipeIntegration) Accounts() ([]*domain.Account, error) {
	var accounts []*domain.Account
	account := domain.Account{ID: 1, Name: "Self"}
	accounts = append(accounts, &account)
	return accounts, nil
}

// Map Github repos to projects
func (s *GitHubPipeIntegration) Projects() ([]*domain.Project, error) {
	repos, _, err := s.client().Repositories.List(context.Background(), "", nil)
	if err != nil {
		return nil, err
	}
	var projects []*domain.Project
	for _, object := range repos {
		project := domain.Project{
			Active:    true,
			Name:      *object.Name,
			ForeignID: strconv.FormatInt(*object.ID, 10),
		}
		projects = append(projects, &project)
	}
	return projects, nil
}

func (s *GitHubPipeIntegration) SetSince(*time.Time) {}

func (s *GitHubPipeIntegration) SetParams([]byte) error {
	return nil
}

func (s *GitHubPipeIntegration) Users() ([]*domain.User, error) {
	return []*domain.User{}, nil
}

func (s *GitHubPipeIntegration) Clients() ([]*domain.Client, error) {
	return []*domain.Client{}, nil
}

func (s *GitHubPipeIntegration) Tasks() ([]*domain.Task, error) {
	return []*domain.Task{}, nil
}

func (s *GitHubPipeIntegration) TodoLists() ([]*domain.Task, error) {
	return []*domain.Task{}, nil
}

func (s *GitHubPipeIntegration) ExportTimeEntry(*domain.TimeEntry) (int, error) {
	return 0, nil
}

func (s *GitHubPipeIntegration) client() *github.Client {
	t := &oauth.Transport{Token: &s.token}
	return github.NewClient(t.Client())
}
