default: test

build:
	go generate ./...

test:
	go clean -testcache
	go test ./logs ./traces ./metrics -coverprofile=coverage.out
	go tool cover -html=coverage.out