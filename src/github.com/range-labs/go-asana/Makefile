test: test-deps
	go test asana/

cover: test-deps
	@go test -coverprofile=cover.out asana
	@go tool cover -html=cover.out

test-deps:
	@mkdir -p src
	@if [ ! -e "src/asana" ]; then ln -s ../asana src; fi
