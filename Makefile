CONF ?= .env
default: build
build:
	CGO_ENABLED=0 GOOS=linux go build -o ./device-pinger
run:
	go run .
tidy:
	go mod tidy
docker-build:
	docker build --tag device-pinger .
docker-run:
	docker run --env-file=$(CONF) device-pinger