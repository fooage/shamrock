#!/bin/bash

# set script file and proto file path
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" >/dev/null 2>&1 && pwd)"
PROTO_DIR="$SCRIPT_DIR/../proto"
GEN_DIR="$SCRIPT_DIR/../proto/proto_gen"

# ensure generate directory exists
mkdir -p $GEN_DIR

# generate Go files from proto files
protoc --proto_path=$PROTO_DIR --go_out=$GEN_DIR \
  --go-grpc_out=$GEN_DIR $(find $PROTO_DIR -name *.proto)

echo "go files generated successfully in $GEN_DIR"
