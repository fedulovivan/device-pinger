FROM golang:1.22 AS builder
COPY go.mod go.sum ./
RUN go mod download
COPY internal internal
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /device-pinger

FROM scratch
COPY --from=builder /device-pinger /device-pinger
CMD ["/device-pinger"]