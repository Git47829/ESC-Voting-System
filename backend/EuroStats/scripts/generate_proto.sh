#!/usr/bin/env sh
set -e

PROTO_SRC=/app/proto
PROTO_OUT=/app/src/proto

echo "Generating protobuf files from ${PROTO_SRC}/*.proto..."

python -m grpc_tools.protoc \
    -I "${PROTO_SRC}" \
    --python_out="${PROTO_OUT}" \
    --grpc_python_out="${PROTO_OUT}" \
    "${PROTO_SRC}/votes.proto"

echo "Patching absolute import in votes_pb2_grpc.py to relative import..."

sed -i \
    's/^import votes_pb2 as votes__pb2$/from . import votes_pb2 as votes__pb2/' \
    "${PROTO_OUT}/votes_pb2_grpc.py"

echo "Proto generation complete."
