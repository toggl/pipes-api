APPNAME=pipes-api
BUGSNAG_API_KEY := '0dd013f86222a229cc6116df663900c8'
BUGSNAG_DEPLOY_NOTIFY_URL := 'https://notify.bugsnag.com/deploy'

test: inittestdb
	go test -v -race -cover

test-integration: inittestdb
	source config/asana_test_account.sh; source config/github_test_account.sh; \
		go test -v -race -cover -tags=integration

inittestdb:
	psql -c 'DROP database pipes_test;' -U postgres
	psql -c 'CREATE database pipes_test;' -U postgres
	psql pipes_test < db/schema.sql

run:
	mkdir -p bin
	cp -r config bin/
	go build -race -o bin/$(APPNAME) && ./bin/$(APPNAME)

.PHONY: dist
dist:
	rm -rf dist
	mkdir -p dist
	cp -r config dist/
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o dist/$(APPNAME)

build:
	go build

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
