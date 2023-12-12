default: test

build:
	go generate ./...

test:
	go clean -testcache
	go test ./logs ./traces ./metrics ./health -coverprofile=coverage.out
	go tool cover -html=coverage.out