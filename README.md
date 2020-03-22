# Pipes API

Backend for the Toggl Pipes project.

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
