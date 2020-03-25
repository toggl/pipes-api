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

## Tests
to run pipes test: `make test`

to run integrations tests:
	- get a token: https://app.asana.com/0/developer-console
	- create a file: `./config/asana_test_account.sh`
	- add your personal token: `export ASANA_PERSONAL_TOKEN=my_token...`
	- run `make test-integration`

## Integrations

### Asana

**WORKS FINE**
To register application use this link: https://app.asana.com/0/developer-console

### GitHub

**WORKS FINE**
To register application use this link: https://github.com/settings/developers

### BaseCamp

**DOES NOT WORK**
NOTE: Integration will work only for BaseCamp 2 account (Codename: bcx). It also use [Basecamp API v2](https://github.com/basecamp/bcx-api/) which is DEPRECATED.
To register application use this link: https://launchpad.37signals.com/integrations

### Freshbooks

**DOES NOT WORK**
Login Form for classic version is here: https://classic.freshbooks.com/
NOTE: Integration supports only Freshbook Classic. It use [Freshbooks Classic](https://www.freshbooks.com/classic-api) API which is DEPRECATED.

### Toggl.Plan (Teamweek)

**DOES NOT WORK**
Outdated, because product name has been changed and API has been moved to different domain.
