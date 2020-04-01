# Pipes API

Backend for the [Toggl Pipes](https://support.toggl.com/en/collections/1148668-import-export#integrations-via-toggl-pipes) project. 
UI can be found in [toggl/pipes-ui](https://github.com/toggl/pipes-ui) repository.

[![CI](https://github.com/toggl/pipes-api/workflows/CI/badge.svg)](https://github.com/toggl/pipes-api/actions?query=workflow%3ACI)

Basically this is just job scheduler with REST-API. 
It schedules job to do synchronization to a third party service.
Pipes-api will fetch data then send them to pipes endpoints in "Toggl API".

**THIS PROJECT WAS FULLY REFACTORED AT March 2020.**

To see original source code use [legacy](https://github.com/toggl/pipes-api/tree/legacy) branch.

## Requirements

* [goenv](https://github.com/syndbg/goenv)
    * [Go 1.14](http://golang.org/)
* [PostgreSQL 9.6](http://www.postgresql.org/) (for development you can use provided Docker file)

## Getting Started

* Clone the repo `git@github.com:toggl/pipes-api.git`
* Copy configuration files `cp -r config-sample config`
* Fill in needed oauth tokens and URL-s under config json files
* Start the server with `make run`

### Architecture

- `pkg` - Stores all abstract business and application logic.
- `internal` - Stores all infrastructure packages.

## Testing

```bash
# First make testing database works. You can use provided Docker container for that:
$ make docker-services-up

# Then just run tests. You also can use Goland IDE for testing:
$ make test

# To generate mocks from source code use:
$ make mocks
```

- To generate Mocks, you should have [mockery](https://github.com/syndbg/goenv) installed.
- In case there is no database installed, all database related tests will be **SKIPPED**.
- All database related tests uses `PIPES_API_POSTGRES_DSN` environment variable to customize database connection string.

## Supported Integrations

### [Asana](https://asana.com)

**WORKS FINE**

To register application use this link: https://app.asana.com/0/developer-console

### [GitHub](https://github.com)

**WORKS FINE**

To register OAuth2 application: https://github.com/settings/developers

### [Toggl.Plan](https://plan.toggl.com) (ex. TeamWeek)

**WORKS FINE**

To register OAuth2 application: https://developers.plan.toggl.com/applications

### [BaseCamp 2](https://basecamp.com/2)

**WORKS FINE**

To register OAuth2 application: https://launchpad.37signals.com/integrations
To register test BaseCamp 2 account: https://billing.37signals.com/bcx/trial/signup/

### [FreshBooks Classic](https://classic.freshbooks.com/)

**DOES NOT WORK**
Login Form for classic version is here: https://classic.freshbooks.com/
NOTE: Integration supports only Freshbook Classic. It use [Freshbooks Classic](https://www.freshbooks.com/classic-api) API which is DEPRECATED.
