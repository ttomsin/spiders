#!/usr/bin/env bash
# Local multi-platform build script for xsai
set -e

VERSION="${1:-v1.0.0}"
DIST_DIR="dist"
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "dev")
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

mkdir -p "$DIST_DIR"

echo "🚀 Building xsai $VERSION ($COMMIT) across all targets..."

TARGETS=(
  "windows/amd64/.exe"
  "windows/arm64/.exe"
  "linux/amd64/"
  "linux/arm64/"
  "darwin/amd64/"
  "darwin/arm64/"
)

for target in "${TARGETS[@]}"; do
  IFS="/" read -r OS ARCH EXT <<< "$target"
  OUT_NAME="xsai-${OS}-${ARCH}${EXT}"
  OUT_PATH="${DIST_DIR}/${OUT_NAME}"

  echo "  -> Building ${OUT_NAME}..."
  GOOS="${OS}" GOARCH="${ARCH}" CGO_ENABLED=0 go build \
    -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${BUILD_DATE}" \
    -o "${OUT_PATH}" ./cmd/x-spider-ai

  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "${OUT_PATH}" | awk '{print $1}' > "${OUT_PATH}.sha256"
  elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "${OUT_PATH}" | awk '{print $1}' > "${OUT_PATH}.sha256"
  fi
done

echo "✅ All targets compiled successfully into ./${DIST_DIR}/!"
ls -la "${DIST_DIR}"
