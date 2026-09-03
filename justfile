default:
    @just --list

run:
    go run ./cmd/waypoint

build:
    mkdir -p build
    go build -o build/waypoint ./cmd/waypoint

test:
    go test ./...

test-verbose:
    go test -v ./...

test-race:
    go test -race ./...

fmt:
    gofmt -w .

vet:
    go vet ./...

lint:
    golangci-lint run

check: fmt vet test

clean:
    rm -rf build/
    go clean

tidy:
    go mod tidy

deps:
    go mod download

generate:
    go generate ./...

sqlc-generate:
    sqlc generate

sqlc-vet:
    sqlc vet

cover:
    go test -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out

cover-html:
    go test -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out

dev:
    go run ./cmd/waypoint
