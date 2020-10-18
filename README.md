# handlergen

[![Actions Status](https://github.com/jungwinter/handlergen/workflows/ci/badge.svg)](https://github.com/jungwinter/handlergen/actions) ![Golang Badge](https://badgen.net/badge/Language/Go/cyan) ![GRPC Badge](https://badgen.net/badge/Use/gRPC/blue)

Generate pre-defined golang grpc handlers with test suite from protobuf file.

1. Generate snake cased handler files with test (`xxx_handler.go`, `xxx_handler_test.go`)
1. Fill empty handler and [test suite](https://godoc.org/github.com/stretchr/testify/suite)

## Install
  1. Clone the repo
  1. Run `go install`
  1. Now you can run `handlergen`

## Usage
```
Usage of handlergen:
  -i <path>
    	protobuf file path to generate
  -o <dir>
    	write output to <dir>
```

### Example

```sh
$ handlergen -i ./sample.proto -o ./
```
