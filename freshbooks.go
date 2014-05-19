package main

import (
	"fmt"
)

type FreshbooksService struct {
	workspaceID int
	AccountID   int
	AccessToken string
}

func (s *FreshbooksService) Name() string {
	return "freshbooks"
}

func (s *FreshbooksService) WorkspaceID() int {
	return s.workspaceID
}

func (s *FreshbooksService) keyFor(objectType string) string {
	return fmt.Sprintf("freshbooks:%s", objectType)
}

func (s *FreshbooksService) setAuthData(a *Authorization) {
}

func (s *FreshbooksService) setAccount(accountID int) {
	s.AccountID = accountID
}

func (s *FreshbooksService) Accounts() ([]*Account, error) {
	return nil, nil
}

func (s *FreshbooksService) Users() ([]*User, error) {
	return nil, nil
}

func (s *FreshbooksService) Projects() ([]*Project, error) {
	return nil, nil
}

func (s *FreshbooksService) TodoLists() ([]*Task, error) {
	return nil, nil
}

func (s *FreshbooksService) Tasks() ([]*Task, error) {
	return nil, nil
}
