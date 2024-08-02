CONF ?= .env
NAME ?= device-pinger
GIT_REV ?= $(shell git rev-parse --short HEAD)

default: build

.PHONY: build
build: lint test
	CGO_ENABLED=0 go build -o $(NAME)

.PHONY: run
run:
	go run .

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: lint
lint:
	golangci-lint run

.PHONY: test
test:
	go test -cover -race -count 1 ./...

.PHONY: docker-build
docker-build:
	DOCKER_CLI_HINTS=false docker build --label "git.revision=${GIT_REV}" --tag $(NAME) .

.PHONY: docker-down
docker-down:
	docker stop $(NAME) && docker rm $(NAME)

.PHONY: docker-up
docker-up:
	docker run -d --env-file=$(CONF) --name=$(NAME) $(NAME)

.PHONY: docker-images
docker-images:
	docker images | grep $(NAME)

.PHONY: docker-logs
docker-logs:
	docker logs --follow $(NAME)

.PHONY: clean
clean:
	rm -f ./$(NAME)