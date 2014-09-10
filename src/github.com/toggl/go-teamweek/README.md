go-teamweek
===========

[Go](http://golang.org) library for accessing the [Teamweek API](https://github.com/Teamweek/teamweek/wiki)

[![Build Status](https://travis-ci.org/toggl/go-teamweek.svg?branch=master)](https://travis-ci.org/toggl/go-teamweek)
[![Coverage Status](https://coveralls.io/repos/toggl/go-teamweek/badge.png?branch=master)](https://coveralls.io/r/toggl/go-teamweek?branch=master)

### Usage ###

```go
import "github.com/toggl/go-teamweek/teamweek"
```

Create a new Teamweek client instance, then use provided methods on the client to
access the API. For example, to list all accounts:

```go
client := teamweek.NewClient(nil)
accounts, err := client.ListAccounts()
```

### Authentication ###

The go-teamweek library does not directly handle authentication. Instead, when
creating a new client, pass an `http.Client` that can handle authentication for
you. The easiest way to do this is using the [goauth2][] library, but you can
always use any other library that provides an `http.Client`. If you have an OAuth2
access token, you can use it with the goauth2 using:

```go
t := &oauth.Transport{
  Token: &oauth.Token{AccessToken: "... your access token ..."},
}

client := teamweek.NewClient(t.Client())

// Get authenticated user's profile information
profile, err := client.Profile()
```

See the [goauth2 docs][] for complete instructions on using that library.

[goauth2]: https://code.google.com/p/goauth2/
[goauth2 docs]: http://godoc.org/code.google.com/p/goauth2/oauth
