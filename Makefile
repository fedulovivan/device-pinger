CONF ?= .env
NAME ?= device-pinger
GIT_REV ?= $(shell git rev-parse --short HEAD)

default: build

build: lint test
	CGO_ENABLED=0 go build -o $(NAME)

run-norace:
	go run .

run:
	GORACE="halt_on_error=1" go run -race .

tidy:
	go mod tidy

lint:
	golangci-lint run

test:
	go test -cover -race -count 1 ./...

docker-build:
	DOCKER_CLI_HINTS=false docker build --label "git.revision=${GIT_REV}" --tag $(NAME) .

docker-down:
	docker stop $(NAME) && docker rm $(NAME)

docker-up:
	docker run -d --env-file=$(CONF) -p 2112:2112 --name=$(NAME) $(NAME)

docker-images:
	docker images | grep $(NAME)

docker-logs:
	docker logs --follow $(NAME)

clean:
	rm -f ./$(NAME)