FROM golang:1.22 AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY internal internal
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /build/device-pinger

FROM scratch
COPY --from=builder /build/device-pinger /device-pinger
CMD ["/device-pinger"]