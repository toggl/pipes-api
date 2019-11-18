go-asana
========

[![Build Status](https://travis-ci.org/tambet/go-asana.svg?branch=master)](https://travis-ci.org/tambet/go-asana)
[![GoDoc](https://godoc.org/github.com/tambet/go-asana/asana?status.svg)](https://godoc.org/github.com/tambet/go-asana/asana)
[![Coverage Status](https://coveralls.io/repos/tambet/go-asana/badge.png?branch=master)](https://coveralls.io/r/tambet/go-asana?branch=master)

[Go](https://golang.org) library for accessing the [Asana API](https://asana.com/developers/documentation/getting-started/overview).

**Note:** go-asana is currently in development, so its API may have slightly breaking changes if we find better ways of doing things.

### Usage ###

```go
import "github.com/tambet/go-asana/asana"
```

Create a new Asana client instance, then use provided methods on the client to
access the API. For example, to list all workspaces:

```go
client := asana.NewClient(nil)
workspaces, err := client.ListWorkspaces()
```

### Authentication ###

The go-asana library does not directly handle authentication. Instead, when
creating a new client, pass an `http.Client` that can handle authentication for
you. The easiest way to do this is using the [goauth2][] library, but you can
always use any other library that provides an `http.Client`. If you have an OAuth2
access token, you can use it with the goauth2 using:

```go
t := &oauth.Transport{
  Token: &oauth.Token{AccessToken: "... your access token ..."},
}

client := asana.NewClient(t.Client())

// List all projects for the authenticated user
projects, err := client.ListProjects(opt)
```

See the [goauth2 docs][] for complete instructions on using that library.

[goauth2]: https://github.com/golang/oauth2
[goauth2 docs]: https://godoc.org/golang.org/x/oauth2
