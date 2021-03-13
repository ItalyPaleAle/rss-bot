# Protobuf definitions

## Requirements

- github.com/golang/protobuf/protoc-gen-go
- github.com/mitchellh/protoc-gen-go-json

```sh
cd $GOPATH
go get -u google.golang.org/protobuf/cmd/protoc-gen-go
go get -u google.golang.org/grpc/cmd/protoc-gen-go-grpc
go get -u github.com/mitchellh/protoc-gen-go-json
```

## Build

```sh
protoc \
  *.proto \
  --go_out=. \
  --go_opt=paths=source_relative \
  --go-json_out=. \
  --go-grpc_out=. \
  --go-grpc_opt=paths=source_relative
```
