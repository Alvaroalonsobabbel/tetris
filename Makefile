check: lint test

test:
	@go test ./...

lint:
	@golangci-lint run

run-tetris: mod
	@go run main.go

build-tetris: mod
	@CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ./bin/tetris
	@chmod +x ./bin/tetris

mod:
	@go mod download

proto:
	@protoc --go_out=./ --go_opt=paths=source_relative --go-grpc_out=./server --go-grpc_opt=paths=source_relative ./server/server.proto
