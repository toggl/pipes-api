go-basecamp
=====
This project implements a [Go](http://golang.org) client library for the [Basecamp API](https://github.com/basecamp/bcx-api/)

Installing
----------
Run
```bash
go get github.com/toggl/go-basecamp
```

Example usage:
```go
package main

import (
  "github.com/toggl/go-basecamp"
  "log"
)

func main() {
  var (
    err      error
    accounts []*basecamp.Account
    projects []*basecamp.Project
    people   []*basecamp.Person
  )

  c := basecamp.Client{AccessToken: "<PUT YOUR AUTH TOKEN HERE>"}

  if accounts, err = c.GetAccounts(); err != nil {
    log.Printf("ERROR %q", err)
    return
  }
  if projects, err = c.GetProjects(accounts[0].Id); err != nil {
    log.Printf("ERROR %q", err)
    return
  }

  if people, err = c.GetPeople(accounts[0].Id); err != nil {
    log.Printf("ERROR %q", err)
    return
  }

  log.Println(projects)
  log.Println(people)
}
```
