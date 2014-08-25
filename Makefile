APPNAME=pipes-api
export GOPATH=$(shell pwd)

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

dist_dir:
	if [ ! -d "dist" ]; then mkdir -p dist; fi

dist: clean dist_dir
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

