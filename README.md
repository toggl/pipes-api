# Pipes API

Backend for the Toggl Pipes project.

[![CI](https://github.com/toggl/pipes-api/workflows/CI/badge.svg)](https://github.com/toggl/pipes-api/actions?query=workflow%3ACI)

Basically this is just job scheduler with REST-API. 
It schedules job to do synchronization to a third party service.
Pipes-api will fetch data then send them to pipes endpoints in "Toggl API".

THIS IS REFACTORED VERSION. To original version see [legacy](https://github.com/toggl/pipes-api/tree/legacy) branch.

## Requirements

* [goenv](https://github.com/syndbg/goenv)
* [Go 1.13.8](http://golang.org/)
* [PostgreSQL 9.6](http://www.postgresql.org/)
* [mockery](https://github.com/syndbg/goenv) - for generating Mocks

## Getting Started

* Clone the repo `git@github.com:toggl/pipes-api.git`
* Copy configuration files `cp -r config-sample config`
* Fill in needed oauth tokens and URL-s under config json files
* Start the server with `make run`

## Testing

```bash
# Firstly make testing database. This can be done only once.
$ make init-test-db

# Then just run tests. You also can use Goland IDE for testing.
$ make test
```

## Integrations

### Asana

**WORKS FINE**

To register application use this link: https://app.asana.com/0/developer-console

### GitHub

**WORKS FINE**

To register OAuth2 application: https://github.com/settings/developers

### Toggl.Plan

**WORKS FINE**

To register OAuth2 application: https://developers.plan.toggl.com/applications


### BaseCamp 2

**WORKS FINE**

To register OAuth2 application: https://launchpad.37signals.com/integrations
To register test BaseCamp 2 account: https://billing.37signals.com/bcx/trial/signup/

### Freshbooks

**DOES NOT WORK**
Login Form for classic version is here: https://classic.freshbooks.com/
NOTE: Integration supports only Freshbook Classic. It use [Freshbooks Classic](https://www.freshbooks.com/classic-api) API which is DEPRECATED.
