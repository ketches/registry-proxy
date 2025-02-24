# Build the application from source
FROM golang:1.24 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -o ./bin/registry-proxy

# Deploy the application binary into a lean image
FROM alpine:latest

WORKDIR /run

COPY --from=builder /app/bin/registry-proxy ./registry-proxy

ENTRYPOINT ["./registry-proxy"]