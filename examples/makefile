default: build

BIN_DIR=./bin/
BIN_NAME=monex

build:
	CGO_ENABLED=0 go build -ldflags "-s -w" -o $(BIN_DIR)$(BIN_NAME) .

img:
	docker build --no-cache --build-arg BUILD_DATE="$(date +%Y%m%d)" -t twistingmercury/monex:latest .
	docker system prune -f

run: img
	docker compose up

client:
	python3 ./testclient/client.py