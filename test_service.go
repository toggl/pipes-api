package main

import (
	"encoding/json"
	"fmt"

	"code.google.com/p/goauth2/oauth"
)

const TestServiceName = "test_service"
const p1Name = "Without surrounding spaces"
const p2Name = "Trailing space "
const p3Name = " Leading space"
const p4Name = " Leading and trailing spaces "
const p5Name = " "

type TestService struct {
	emptyService
	workspaceID int
	token       oauth.Token
}

func (s *TestService) Name() string {
	return TestServiceName
}

func (s *TestService) WorkspaceID() int {
	return s.workspaceID
}

func (s *TestService) keyFor(pipeID string) string {
	return fmt.Sprintf("test:%s", pipeID)
}

func (s *TestService) setAuthData(b []byte) error {
	if err := json.Unmarshal(b, &s.token); err != nil {
		return err
	}
	return nil
}

func (s *TestService) Projects() ([]*Project, error) {
	var ps []*Project
	ps = append(ps, &Project{Name: p1Name})
	ps = append(ps, &Project{Name: p2Name})
	ps = append(ps, &Project{Name: p3Name})
	ps = append(ps, &Project{Name: p4Name})
	ps = append(ps, &Project{Name: p5Name})
	return ps, nil
}
