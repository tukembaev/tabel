# syntax=docker/dockerfile:1
# OPTIONAL — local laptop only. Production = Linux binaries + systemd (see README, deploy/).

FROM golang:1.26-alpine AS build
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/tabel-server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/tabel-worker ./cmd/worker

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/tabel-server /app/tabel-server
COPY --from=build /out/tabel-worker /app/tabel-worker
COPY migrations /app/migrations
EXPOSE 8080
CMD ["/app/tabel-server"]
