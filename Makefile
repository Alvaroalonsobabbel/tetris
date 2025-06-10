.PHONY: check test lint run-tetris build-tetris mod proto

check: lint test

test:
	@go test -race ./...

lint:
	@golangci-lint run

run-tetris: mod
	@go run main.go

build-tetris: mod
	@CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ./bin/tetris
	@chmod +x ./bin/tetris

run-server: mod
	@go run cmd/server/main.go

mod:
	@go mod download

proto:
	@protoc --go_out=./ --go_opt=paths=source_relative --go-grpc_out=./ --go-grpc_opt=paths=source_relative ./proto/server.proto
