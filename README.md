## Pipes API

Backend for the Toggl Pipes project. Currently in development.

[![Build Status](https://travis-ci.org/toggl/pipes-api.svg?branch=master)](https://travis-ci.org/toggl/pipes-api)

## Installation
    $ cp -r config-sample config
    
Fill in needed oauth tokens and URL-s under config/*.json

## Usage
    make run

## Creating your own pipe
Each new service must implement [Service][2] inteface


[1]: https://github.com/toggl/pipes-ui
[2]: https://github.com/toggl/pipes-api/blob/master/service.go
