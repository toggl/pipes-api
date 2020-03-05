APPNAME=pipes-api
BUGSNAG_API_KEY:=0dd013f86222a229cc6116df663900c8
BUGSNAG_DEPLOY_NOTIFY_URL:=https://notify.bugsnag.com/deploy
REVISION:=$(shell git rev-parse HEAD)
REPOSITORY:=git@github.com:toggl/pipes-api.git


all: build test

test: inittestdb
	go test -race -cover ./pkg/...

test-integration: inittestdb
	source config/test_accounts.sh && go test -v -race -cover -tags=integration ./pkg/...

inittestdb:
	psql -c 'DROP database pipes_test;' -U postgres
	psql -c 'CREATE database pipes_test;' -U postgres
	psql pipes_test < db/schema.sql

mocks:
	mockery -dir ./pkg/pipe -output ./pkg/pipe/mocks -case underscore -all
	mockery -dir ./pkg/oauth -output ./pkg/oauth/mocks -case underscore -all
	mockery -dir ./pkg/integrations -output ./pkg/integrations/mocks -case underscore -all

run:
	mkdir -p bin
	cp -r config bin/
	go build -race -o bin/$(APPNAME) ./cmd/pipes-api && ./bin/$(APPNAME)

.PHONY: dist
dist:
	rm -rf dist
	mkdir -p dist
	cp -r config dist/
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/$(APPNAME) ./cmd/pipes-api

build:
	go build -o bin/$(APPNAME) ./cmd/pipes-api

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
