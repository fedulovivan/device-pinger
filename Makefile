default:
	go build
run:
	go run .
tidy:
	go mod tidy
docker-build:
	docker build --tag device-pinger .
docker-run:
	docker run --env-file=.env device-pinger