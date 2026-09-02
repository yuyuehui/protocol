#!/usr/bin/env bash
set -euo pipefail

# ===========================================================================
# Generator versions must match what produced the checked-in *.pb.go, or every
# file in the repo flips and buries the real diff. Known differences:
#   protoc-gen-go-grpc 1.5.x emits status.Errorf(...), 1.6.0 emits status.Error(...)
#   protoc-gen-go 1.36.8 vs 1.36.11 render line comments differently (//// vs // ///)
# Do not regenerate with a different version and commit the result.
#
#   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.6.0
#   go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
#
# Keep this file in sync with gen.cmd: same PROTO_NAMES, same versions, same
# omitempty strip. They must produce byte-identical output.
# ===========================================================================
WANT_GRPC="1.6.0"
WANT_GO="v1.36.11"

HAVE_GRPC="$(protoc-gen-go-grpc --version 2>&1 | awk '{print $2}')"
if [ "${HAVE_GRPC}" != "${WANT_GRPC}" ]; then
    echo "[gen] protoc-gen-go-grpc version mismatch: have \"${HAVE_GRPC}\", want \"${WANT_GRPC}\""
    echo "[gen]   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v${WANT_GRPC}"
    exit 1
fi

HAVE_GO="$(protoc-gen-go --version 2>&1 | awk '{print $2}')"
if [ "${HAVE_GO}" != "${WANT_GO}" ]; then
    echo "[gen] protoc-gen-go version mismatch: have \"${HAVE_GO}\", want \"${WANT_GO}\""
    echo "[gen]   go install google.golang.org/protobuf/cmd/protoc-gen-go@${WANT_GO}"
    exit 1
fi

# Keep this list in sync with gen.cmd; add new .proto dirs to both.
PROTO_NAMES=(
    "auth"
    "conversation"
    "errinfo"
    "relation"
    "group"
    "jssdk"
    "msg"
    "msggateway"
    "push"
    "rtc"
    "sdkws"
    "third"
    "user"
    "statistics"
    "wrapperspb"
    "oa"
    "livekit_meeting"
    "meeting_room"
    "schedule"
    "egress"
    "email"
    "call"
)

for name in "${PROTO_NAMES[@]}"; do
  if ! protoc --go_out="./${name}" --go_opt="module=github.com/openimsdk/protocol/${name}" "${name}/${name}.proto"; then
      echo "error processing ${name}.proto (go_out)"
      exit 1
  fi
done

# Generate Go gRPC code.
for name in "${PROTO_NAMES[@]}"; do
  if ! protoc --go-grpc_out="./${name}" --go-grpc_opt="module=github.com/openimsdk/protocol/${name}" "${name}/${name}.proto"; then
      echo "error processing ${name}.proto (go-grpc_out)"
      exit 1
  fi
done

# Strip omitempty from json tags (this fork serializes zero values).
#
# The match must be bare ",omitempty", not ',omitempty"`' -- map fields render as
#   json:"maxSeqs,omitempty" protobuf_key:"..." protobuf_val:"..."
# so the tag does not end at the omitempty and a backtick-anchored pattern misses
# all 26 of them, leaving gen.sh and gen.cmd with divergent output.
# Every ",omitempty" in generated code is the tail of a json tag value, so an
# unanchored strip is safe.
#
# Repo-wide by design: this also covers openmeeting/*, which gen_openmeeting.sh
# generates. The fork convention is "no omitempty anywhere", not "no omitempty in
# what this script regenerates".
if [ "$(uname -s)" == "Darwin" ]; then
    find . -type f -name '*.pb.go' -exec sed -i '' 's/,omitempty//g' {} +
else
    find . -type f -name '*.pb.go' -exec sed -i 's/,omitempty//g' {} +
fi
