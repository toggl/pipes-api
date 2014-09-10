test: test-deps
	go test teamweek/

cover: test-deps
	@go test -coverprofile=cover.out teamweek
	@go tool cover -html=cover.out

test-deps:
	@mkdir -p src
	@if [ ! -e "src/teamweek" ]; then ln -s ../teamweek src; fi
