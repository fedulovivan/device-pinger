CONF ?= .env
NAME ?= device-pinger

default: build

.PHONY: build
build: test
	CGO_ENABLED=0 go build -o ./$(NAME)

.PHONY: run
run:
	go run .

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: test
test:
	go test -cover -race -count 1 ./...

.PHONY: docker-build
docker-build: test
	DOCKER_CLI_HINTS=false docker build --tag $(NAME) .

.PHONY: docker-run
docker-run:
	docker run --env-file=$(CONF) $(NAME)

.PHONY: clean
clean:
	rm -f ./$(NAME)