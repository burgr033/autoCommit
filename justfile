
BINARY_NAME := "autoCommit"
PACKAGE_NAME := "github.com/burgr033/autoCommit"

default: help

# init the project
init:
  @echo init project
  go mod init {{PACKAGE_NAME}}
  git init .
  mkdir -p {cmd/{{BINARY_NAME}},internal/,pkg/}

# runs golangci-lint
lint:
  @echo Running linter...
  golangci-lint run || exit 1

# cleans up, removes binary, exe out and html files
clean:
  @echo Cleaning up...
  @rm -f *.exe coverage.out coverage.html {{BINARY_NAME}}

# runs tests
test:
  @echo Running tests...
  go test -v -race -cover ./...

# runs tests with test coverage enabled
test-coverage:
  @echo Running tests with coverage
  go test -v -race -coverprofile=coverage.out ./...
  go tool cover -html=coverage.out -o coverage.html

# opens up the coverage report (via xdg-open)
view-test-coverage: test-coverage
  @echo Showing test coverage
  @xdg-open coverage.html &

# builds the application {{BINARY_NAME}}
build:
  @echo building application
  go build -o {{BINARY_NAME}} cmd/{{BINARY_NAME}}/main.go

# runs the application {{BINARY_NAME}}
run: build
  ./{{BINARY_NAME}}

release:
  goreleaser

# help message
help:
  @just -l

