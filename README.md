# Pipes API

Backend for the Toggl Pipes project. Currently in development.

[![Build Status](https://travis-ci.org/toggl/pipes-api.svg?branch=master)](https://travis-ci.org/toggl/pipes-api)

## Requirements

* Go 1.3 - [http://golang.org/](http://golang.org/)
* PostgreSQL 9.3 - [http://www.postgresql.org/](http://www.postgresql.org/)

## Getting Started
* Clone the repo `git@github.com:toggl/pipes-api.git`
* Copy configuration files `cp -r config-sample config`
* Fill in needed oauth tokens and URL-s under config json files
* Start the server with `make run`

## Creating a new pipe
Each new service must implement [Service][2] inteface


[1]: https://github.com/toggl/pipes-ui
[2]: https://github.com/toggl/pipes-api/blob/master/service.go
