APPNAME=pipes-api
BUGSNAG_API_KEY:=0dd013f86222a229cc6116df663900c8
BUGSNAG_DEPLOY_NOTIFY_URL:=https://build.bugsnag.com/
APP_REVISION:=$(shell git rev-parse HEAD)
APP_VERSION:=$(shell git describe --tags --abbrev=0 HEAD)
BUILD_TIME := $(shell date '+%Y%m%d-%H:%M:%S')
BUILD_AUTHOR := $(shell git config --get user.email)
REPOSITORY:=git@github.com:toggl/pipes-api.git
LD_FLAGS:=-X 'main.Version=$(APP_VERSION)' -X 'main.Revision=$(APP_REVISION)' -X 'main.BuildTime=$(BUILD_TIME)' -X 'main.BuildAuthor=$(BUILD_AUTHOR)'

# Used for debugging purposes. To have possibility connect to a process with Delve debugger
GC_FLAGS:=all=-N -l

all: init-dev-db init-test-db build test

clean:
	rm -Rf ./bin ./dist ./out

init-test-db:
	psql -c 'DROP DATABASE IF EXISTS pipes_test;' -U postgres
	psql -c 'CREATE DATABASE pipes_test;' -U postgres
	psql pipes_test < db/schema.sql

init-dev-db:
	psql -c 'DROP DATABASE IF EXISTS pipes_development;' -U postgres
	psql -c 'CREATE DATABASE pipes_development;' -U postgres
	psql pipes_development < db/schema.sql

mocks:
	go generate ./internal/... ./pkg/...

test:
	go test -race -cover ./internal/... ./pkg/...

build:
	go build -ldflags="$(LD_FLAGS)" -o bin/$(APPNAME) ./cmd/pipes-api
	go build -o bin/toggl_api_stub ./cmd/toggl_api_stub # This binary needs only for testing purposes. For more information see main.go of this binary.

build-release:
	rm -rf dist
	mkdir -p dist
	cp -r config dist/
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$(LD_FLAGS)" -o dist/$(APPNAME) ./cmd/pipes-api
	cd dist && tar czf pipes-api.tgz pipes-api config && cd ../

run:
	mkdir -p bin
	cp -r config bin/
	go build -ldflags="$(LD_FLAGS)" -gcflags="$(GC_FLAGS)" -race -o bin/$(APPNAME) ./cmd/pipes-api && ./bin/$(APPNAME)

staging: build-release
	rsync -avz -e "ssh -p 22" dist/pipes-api.tgz toggl@appseed.toggl.space:/var/www/office/appseed/pipes-api/staging.tgz && \
	crap staging && \
	curl --silent --show-error --fail --include --request POST --header "Content-Type: application/json" --data-binary "{\"apiKey\": \"$(BUGSNAG_API_KEY)\",\"appVersion\": \"$(APP_VERSION)\",\"builderName\": \"$(BUILD_AUTHOR)\",\"sourceControl\": {\"repository\": \"$(REPOSITORY)\",\"revision\": \"$(APP_REVISION)\"},\"releaseStage\":\"staging\"}" $(BUGSNAG_DEPLOY_NOTIFY_URL)

production: build-release
	rsync -avz -e "ssh -p 22" dist/pipes-api.tgz toggl@appseed.toggl.space:/var/www/office/appseed/pipes-api/production.tgz && \
	crap production && \
	curl --silent --show-error --fail --include --request POST --header "Content-Type: application/json" --data-binary "{\"apiKey\": \"$(BUGSNAG_API_KEY)\",\"appVersion\": \"$(APP_VERSION)\",\"builderName\": \"$(BUILD_AUTHOR)\",\"sourceControl\": {\"repository\": \"$(REPOSITORY)\",\"revision\": \"$(APP_REVISION)\"},\"releaseStage\":\"production\"}" $(BUGSNAG_DEPLOY_NOTIFY_URL)

dependency-graph:
	mkdir -p out
	godepgraph -p code.google.com,github.com/bugsnag,github.com/lib/pq,github.com/tambet,github.com/google \
 		-nostdlib -novendor ./cmd/pipes-api | dot -Tpng -o out/deps.png
