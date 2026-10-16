.PHONY: build build-linux run worker migrate sqlc test install-local

build:
	mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o bin/tabel-server ./cmd/server
	CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o bin/tabel-worker ./cmd/worker

build-linux:
	./scripts/build-linux.sh

run: build
	./bin/tabel-server

worker: build
	./bin/tabel-worker

sqlc:
	sqlc generate

migrate:
	goose -dir migrations postgres "$(DATABASE_URL)" up

test:
	go test ./...

# Local convenience only — production uses systemd binaries, not Docker.
install-local: build
	@echo "Binaries in ./bin — copy to server or run ./scripts/install-server.sh as root on the host."
