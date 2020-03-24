APPNAME=pipes-api
BUGSNAG_API_KEY:=0dd013f86222a229cc6116df663900c8
BUGSNAG_DEPLOY_NOTIFY_URL:=https://notify.bugsnag.com/deploy
APP_REVISION:=$(shell git rev-parse HEAD)
APP_VERSION:=$(shell git describe --tags --abbrev=0 HEAD)
BUILD_TIME := $(shell date '+%Y%m%d-%H:%M:%S')
BUILD_AUTHOR := $(shell git config --get user.email)
REPOSITORY:=git@github.com:toggl/pipes-api.git
LD_FLAGS:=-ldflags="-X 'main.Version=$(APP_VERSION)' -X 'main.Revision=$(APP_REVISION)' -X 'main.BuildTime=$(BUILD_TIME)' -X 'main.BuildAuthor=$(BUILD_AUTHOR)'"

all: init-dev-db build test

test: init-test-db
	go test -race -cover ./pkg/...

test-integration: init-test-db
	source config/test_accounts.sh && go test -v -race -cover -tags=integration ./pkg/...

init-test-db:
	psql -c 'DROP DATABASE IF EXISTS pipes_test;' -U postgres
	psql -c 'CREATE DATABASE pipes_test;' -U postgres
	psql pipes_test < db/schema.sql

init-dev-db:
	psql -c 'DROP DATABASE IF EXISTS pipes_development;' -U postgres
	psql -c 'CREATE DATABASE pipes_development;' -U postgres
	psql pipes_development < db/schema.sql

mocks:
	go generate ./pkg/...

run:
	mkdir -p bin
	cp -r config bin/
	go build $(LD_FLAGS) -gcflags="all=-N -l" -race -o bin/$(APPNAME) ./cmd/pipes-api && ./bin/$(APPNAME)

.PHONY: dist
dist:
	rm -rf dist
	mkdir -p dist
	cp -r config dist/
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LD_FLAGS) -o dist/$(APPNAME) ./cmd/pipes-api

build:
	go build $(LD_FLAGS) -o bin/$(APPNAME) ./cmd/pipes-api
	go build -o bin/toggl_api_stub ./cmd/toggl_api_stub # This binary needs only for testing purposes. For more information see main.go of this binary.

vendor: dist
	cd dist && tar czf pipes-api.tgz pipes-api config

send-vendor-staging: vendor
	rsync -avz -e "ssh -p 22" dist/pipes-api.tgz toggl@appseed.toggl.space:/var/www/office/appseed/pipes-api/staging.tgz

send-vendor-production: vendor
	rsync -avz -e "ssh -p 22" dist/pipes-api.tgz toggl@appseed.toggl.space:/var/www/office/appseed/pipes-api/production.tgz

staging: send-vendor-staging
	crap staging; \
		curl --silent --show-error --fail -X POST -d "apiKey=$(BUGSNAG_API_KEY)&releaseStage=staging&revision=$(REVISION)&repository=$(REPOSITORY)" $(BUGSNAG_DEPLOY_NOTIFY_URL)

production: send-vendor-production
	crap production; \
		curl --silent --show-error --fail -X POST -d "apiKey=$(BUGSNAG_API_KEY)&releaseStage=production&revision=$(REVISION)&repository=$(REPOSITORY)" $(BUGSNAG_DEPLOY_NOTIFY_URL)
