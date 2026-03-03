#!/bin/bash

mkdir -p src/proto

# Generate Python gRPC code from proto file
python3 -m grpc_tools.protoc \
    -I./proto \
    --python_out=./src/proto \
    --grpc_python_out=./src/proto \
    ./proto/votes.proto

# Create __init__.py file if it doesn't exist
touch ./src/proto/__init__.py

echo "Proto files generated successfully!"
