default:
    @just --list

run *args="":
    go run ./cmd/waypoint {{args}}

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
    go fmt .

vet:
    go vet ./...

lint:
    golangci-lint run

check: fmt vet test

clean:
    rm -rf build/
    go clean
