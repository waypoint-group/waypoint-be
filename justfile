default:
    @just --list

run *args="":
    go run ./cmd/waypoint {{args}}

build:
    mkdir -p build
    go build -o build/waypoint ./cmd/waypoint

test-unit *args="":
    go test {{args}} ./...

test *args="":
    go test {{args}} --tags integration ./... 

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
