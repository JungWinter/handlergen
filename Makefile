GOPATH:=$(shell go env GOPATH)

.PHONY: format
## format: format files
format:
	@go get golang.org/x/tools/cmd/goimports
	goimports -w .
	gofmt -s -w .

.PHONY: test
## test: run tests
test:
	@go get github.com/rakyll/gotest
	gotest -p 1 -race -v ./...

.PHONY: coverage
## coverage: run tests with coverage
coverage:
	@go get github.com/rakyll/gotest
	gotest -p 1 -race -coverprofile=coverage.txt -covermode=atomic -v ./...

.PHONY: lint
## lint: check everything's okay
lint:
	@go get github.com/kyoh86/scopelint
	golangci-lint run ./...
	scopelint --set-exit-status ./...
	go mod verify

.PHONY: help
## help: prints this help message
help:
	@echo "Usage: \n"
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':'
