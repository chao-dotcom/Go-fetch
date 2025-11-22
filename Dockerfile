# syntax=docker/dockerfile:1.7

FROM golang:1.23.0 AS builder
WORKDIR /src
COPY go.mod ./
# Download dependencies (this will create go.sum if missing)
RUN go mod download
# Copy source code
COPY . .
# Tidy up go.mod to ensure consistency after code changes
RUN go mod tidy
# Build API
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/api ./cmd/api
# Build Worker
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/worker ./cmd/worker

FROM gcr.io/distroless/base-debian12 AS api
WORKDIR /app
COPY --from=builder /out/api /app/taskqueue-api
EXPOSE 8080
ENV API_ADDR=:8080
ENTRYPOINT ["/app/taskqueue-api"]

FROM gcr.io/distroless/base-debian12 AS worker
WORKDIR /app
COPY --from=builder /out/worker /app/taskqueue-worker
ENTRYPOINT ["/app/taskqueue-worker"]

