APPNAME=pipes-api
export GOPATH=$(shell pwd)

default: clean build fmt lint

vet:
	go vet

run: vet fmt 
	go build -o $(APPNAME) && ./$(APPNAME)

lint:
	go get github.com/golang/lint/golint
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
	rm -f $(APPNAME)
	rm -rf dist
	rm -rf pkg
	rm -rf out
	rm -rf bin
