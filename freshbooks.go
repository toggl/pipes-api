package main

import (
	"encoding/json"
	"fmt"
	"github.com/tambet/oauthplain"
	"github.com/toggl/go-freshbooks"
)

type FreshbooksService struct {
	workspaceID int
	accountName string
	token       oauthplain.Token
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

func (s *FreshbooksService) setParams(b []byte) error {
	return nil
}

func (s *FreshbooksService) setAuthData(b []byte) error {
	if err := json.Unmarshal(b, &s.token); err != nil {
		return err
	}
	return nil
}

func (s *FreshbooksService) Accounts() ([]*Account, error) {
	return nil, nil
}

func (s *FreshbooksService) Users() ([]*User, error) {
	apiClient := freshbooks.NewApi(s.accountName, s.token)
	foreignObjects, err := apiClient.Users()
	if err != nil {
		return nil, err
	}
	var users []*User
	for _, object := range foreignObjects {
		user := User{
			ForeignID: object.UserId,
			Name:      fmt.Sprintf("%s %s", object.FirstName, object.LastName),
			Email:     object.Email,
		}
		users = append(users, &user)
	}
	return users, nil
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
