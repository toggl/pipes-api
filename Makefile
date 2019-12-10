APPNAME=pipes-api
export GOPATH=$(shell pwd)
GOVERSION=$(shell go version | cut -d' ' -f3)
REQUIRED_GOVERSION = $(shell cat .go-version | tr -d '\n')

BUGSNAG_API_KEY := '0dd013f86222a229cc6116df663900c8'
BUGSNAG_DEPLOY_NOTIFY_URL := 'https://notify.bugsnag.com/deploy'

default: clean build fmt

inittestdb:
	- psql -c 'DROP database pipes_test;' -U postgres
	psql -c 'CREATE database pipes_test;' -U postgres
	psql pipes_test < db/schema.sql

vet:
	go vet

test: inittestdb
	go test -v

test-integration: inittestdb
	if [[ ! -f config/asana_test_account.sh ]]; then echo 'please setup pipes-api-conf'; fi
	source config/asana_test_account.sh && go test -v -race -tags=integration

run: vet fmt
	go build -race -o $(APPNAME) && ./$(APPNAME)

bin/golint:
	go get github.com/golang/lint/golint

lint: bin/golint
	bin/golint *.go

check_go_version:
	@if [ ! "${GOVERSION}" = "go${REQUIRED_GOVERSION}" ]; then echo '\nError: invalid go version, check Makefile'; exit 1; fi

dist_dir:
	if [ ! -d "dist" ]; then mkdir -p dist; fi

dist: clean dist_dir check_go_version
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/$(APPNAME)
	cp -r config dist/

fmt:
	go fmt

build:
	go build

clean:
	rm -rf $(APPNAME) dist pkg out bin

bin/errcheck:
	go get github.com/kisielk/errcheck

errcheck: bin/errcheck
	bin/errcheck -ignore 'Close|[wW]rite.*|Flush|Seek|[rR]ead.*|Notify|Rollback'

get:
	go get

DEPENDENCY ?= ""
get-dep:
	@if [ "$(DEPENDENCY)" != "" ]; then \
		rm -fr src/$(DEPENDENCY); \
		export GOPATH=`pwd`; \
		go get $(DEPENDENCY); \
		rm -fr src/$(DEPENDENCY)/.git*; \
	else \
		echo "usage: DEPENDENCY=github.com/login/pkg make get-dep"; \
		exit 1; \
	fi

vendor: dist
	cd dist && tar czf pipes-api.tgz pipes-api config

send-vendor-staging: vendor
	rsync -avz -e "ssh -p 22" dist/pipes-api.tgz toggl@appseed.toggl.space:/var/www/office/appseed/pipes-api/staging.tgz

send-vendor-production: vendor
	rsync -avz -e "ssh -p 22" dist/pipes-api.tgz toggl@appseed.toggl.space:/var/www/office/appseed/pipes-api/production.tgz

staging: send-vendor-staging
	crap staging; \
		curl --silent --show-error --fail -X POST -d "apiKey=$(BUGSNAG_API_KEY)&releaseStage=staging" $(BUGSNAG_DEPLOY_NOTIFY_URL)

production: send-vendor-production
	crap production; \
		curl --silent --show-error --fail -X POST -d "apiKey=$(BUGSNAG_API_KEY)&releaseStage=production" $(BUGSNAG_DEPLOY_NOTIFY_URL)
