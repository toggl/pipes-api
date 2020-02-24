package mock

import (
	"encoding/json"
	"fmt"
	"time"

	"code.google.com/p/goauth2/oauth"

	"github.com/toggl/pipes-api/pkg/integrations"
	"github.com/toggl/pipes-api/pkg/toggl"
)

const ServiceName = "test_service"
const (
	P1Name = "Without surrounding spaces"
	P2Name = "Trailing space "
	P3Name = " Leading space"
	P4Name = " Leading and trailing spaces "
	P5Name = " "
)

type Service struct {
	WorkspaceID int
	token       oauth.Token
}

func (s *Service) Name() string {
	return ServiceName
}

func (s *Service) GetWorkspaceID() int {
	return s.WorkspaceID
}

func (s *Service) KeyFor(pipeID string) string {
	return fmt.Sprintf("test:%s", pipeID)
}

func (s *Service) SetAuthData(b []byte) error {
	if err := json.Unmarshal(b, &s.token); err != nil {
		return err
	}
	return nil
}

func (s *Service) Projects() ([]*toggl.Project, error) {
	var ps []*toggl.Project
	ps = append(ps, &toggl.Project{Name: P1Name})
	ps = append(ps, &toggl.Project{Name: P2Name})
	ps = append(ps, &toggl.Project{Name: P3Name})
	ps = append(ps, &toggl.Project{Name: P4Name})
	ps = append(ps, &toggl.Project{Name: P5Name})
	return ps, nil
}

func (s *Service) SetSince(*time.Time) {}

func (s *Service) SetParams([]byte) error {
	return nil
}

func (s *Service) Accounts() ([]*toggl.Account, error) {
	return []*toggl.Account{}, nil
}

func (s *Service) Users() ([]*toggl.User, error) {
	return []*toggl.User{}, nil
}

func (s *Service) Clients() ([]*toggl.Client, error) {
	return []*toggl.Client{}, nil
}

func (s *Service) Tasks() ([]*toggl.Task, error) {
	return []*toggl.Task{}, nil
}

func (s *Service) TodoLists() ([]*toggl.Task, error) {
	return []*toggl.Task{}, nil
}

func (s *Service) ExportTimeEntry(*toggl.TimeEntry) (int, error) {
	return 0, nil
}

var _ integrations.Integration = (*Service)(nil)
