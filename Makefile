APPNAME=pipes-api
export GOPATH=$(shell pwd)
GOVERSION=$(shell go version | cut -d' ' -f3)

default: clean build fmt

inittestdb:
	- psql -c 'DROP database pipes_test;' -U postgres
	psql -c 'CREATE database pipes_test;' -U postgres
	psql pipes_test < db/schema.sql

vet:
	go vet

test: inittestdb
	go test

run: vet fmt
	go build -o $(APPNAME) && ./$(APPNAME)

bin/golint:
	go get github.com/golang/lint/golint

lint: bin/golint
	bin/golint *.go

check_go_version:
	@if [ ! "${GOVERSION}" = "go1.9.7" ]; then echo '\nError: invalid go version, check Makefile'; exit 1; fi

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
		rm -fr vendor/src/$(DEPENDENCY); \
		export GOPATH=`pwd`/vendor; \
		go get $(DEPENDENCY); \
		rm -fr vendor/src/$(DEPENDENCY)/.git*; \
	else \
		echo "usage: DEPENDENCY=github.com/login/pkg make get-dep"; \
		exit 1; \
	fi
