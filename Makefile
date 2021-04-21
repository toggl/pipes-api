APPNAME=pipes-api
BUGSNAG_API_KEY:=0dd013f86222a229cc6116df663900c8
BUGSNAG_DEPLOY_NOTIFY_URL:=https://notify.bugsnag.com/deploy
REVISION:=$(shell git rev-parse HEAD)
REPOSITORY:=git@github.com:toggl/pipes-api.git

test: inittestdb
	source config/test_accounts.sh && go test -v -race -cover

test-integration: inittestdb
	source config/test_accounts.sh && go test -v -race -cover -tags=integration

.PHONY: config
config:
	@./scripts/update-config.sh

inittestdb:
	psql -c 'DROP database pipes_test;' -U postgres
	psql -c 'CREATE database pipes_test;' -U postgres
	psql pipes_test < db/schema.sql

run:
	mkdir -p bin
	cp -r config bin/
	go build -race -o bin/$(APPNAME) && ./bin/$(APPNAME)

.PHONY: dist
dist: config
	rm -rf dist
	mkdir -p dist
	cp -r config dist/
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/$(APPNAME)

build:
	go build

vendor: dist
	@cd dist && zip -r ${APPNAME}.zip pipes-api config

update-deploy-script:
	@scripts/update-deploy-script.sh

production: vendor update-deploy-script
	@tmp/deploy-script/crap.sh ${APPNAME} dist/${APPNAME}.zip ${REVISION} production

staging: vendor update-deploy-script
	@tmp/deploy-script/crap.sh ${APPNAME} dist/${APPNAME}.zip ${REVISION} staging
