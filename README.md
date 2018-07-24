# Pipes API

Backend for the Toggl Pipes project. Currently in development.

[![Build Status](https://travis-ci.org/toggl/pipes-api.svg?branch=master)](https://travis-ci.org/toggl/pipes-api)

## Requirements

* Go 1.9.7 - [http://golang.org/](http://golang.org/)
* PostgreSQL 9.3 - [http://www.postgresql.org/](http://www.postgresql.org/)

## Getting Started
* Clone the repo `git@github.com:toggl/pipes-api.git`
* Copy configuration files `cp -r config-sample config`
* Fill in needed oauth tokens and URL-s under config json files
* Start the server with `make run`

## Creating a new pipe
Each new service must implement [Service][2] inteface. Currently only services with OAuth 2.0 or OAuth 1.0 "PLAINTEXT" authentication are supported.

## New pipe example
Lets create a pipe to fetch Github repos to Toggl project. First, add the new integration to `config/integrations.json`
```json
{
  "id": "github",
  "name": "Github",
  "auth_type": "oauth2",
  "image": "/images/logo-github.png",
  "link": "https://github.com/toggl/pipes-api",
  "pipes": [
    {
      "id": "projects",
      "name": "Github repos",
      "premium": false,
      "automatic_option": true,
      "description": "Github repos will be imported as Toggl projects. Existing projects are matched by name."
    }
  ]
}
```

Next register a new [Github application](https://github.com/settings/applications) and add the new authorization details to `config/oauth2.json`

```json
"github_development": {
  "ClientId": "<<GITHUB APP CLIENT ID>>",
  "ClientSecret": "<<GITHUB APP CLIENT SECRET>>",
  "AuthURL": "https://github.com/login/oauth/authorize",
  "TokenURL": "https://github.com/login/oauth/access_token",
  "RedirectURL": "<<REDIRECT URL>>"
}
```

For accessing the GitHub API we are going to use the [go-github](https://github.com/google/go-github/) client library.
Install the pacakge with `go get github.com/google/go-github/github`. Now the actual GithubService implementation.

```go
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
```

And finally and the new GithubService to supported services in `service.go`

```go
case "github":
  return Service(&GithubService{workspaceID: workspaceID})
```

All this in one [commit](https://github.com/toggl/pipes-api/commit/9307171c4dcad429cfaa3c406adde7b5ff765340).
We also need to enable the Github integration in [pipes-ui](https://github.com/toggl/pipes-ui/commit/4039a2bc50294d4054d21918f0af627196ff1999) project.

[1]: https://github.com/toggl/pipes-ui
[2]: https://github.com/toggl/pipes-api/blob/master/service.go
