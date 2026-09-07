#!/usr/bin/env bash
# Regenerates generated/types.go from the GraphANN OpenAPI spec.
#
# Usage:
#   scripts/generate-types.sh                  # uses the default spec path
#   GRAPHANN_SPEC=/other/path.yaml scripts/generate-types.sh
#
# Requires oapi-codegen on PATH (or $(go env GOPATH)/bin):
#   go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GO_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Prefer the standalone repository's spec, then the shared monorepo spec.
# An explicit override still wins for development against another spec.
SPEC="${GRAPHANN_SPEC:-}"
if [[ -z "${SPEC}" ]]; then
  if [[ -f "${GO_DIR}/api/openapi/spec.yaml" ]]; then
    SPEC="${GO_DIR}/api/openapi/spec.yaml"
  else
    SPEC="${GO_DIR}/../api/openapi/spec.yaml"
  fi
fi
OUT="${GO_DIR}/generated/types.go"

OAPI_CODEGEN="$(command -v oapi-codegen || true)"
if [[ -z "${OAPI_CODEGEN}" ]]; then
  GOPATH_BIN="$(go env GOPATH)/bin/oapi-codegen"
  if [[ -x "${GOPATH_BIN}" ]]; then
    OAPI_CODEGEN="${GOPATH_BIN}"
  fi
fi
if [[ -z "${OAPI_CODEGEN}" ]]; then
  echo "generate-types.sh: oapi-codegen not found on PATH or in \$(go env GOPATH)/bin" >&2
  echo "install: go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest" >&2
  exit 1
fi

if [[ ! -f "${SPEC}" ]]; then
  echo "generate-types.sh: spec not found at ${SPEC}" >&2
  exit 1
fi

echo "generate-types.sh: ${OAPI_CODEGEN} -generate types -package generated -o ${OUT} ${SPEC}"
"${OAPI_CODEGEN}" -generate types -package generated -o "${OUT}" "${SPEC}"
echo "generate-types.sh: wrote ${OUT} ($(wc -l < "${OUT}" | tr -d ' ') lines)"
