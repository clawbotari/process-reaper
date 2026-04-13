#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
REPO_ROOT=$(cd "${SCRIPT_DIR}/.." && pwd)
IMAGE_NAME="${PROCESS_REAPER_ROCKY9_IMAGE:-process-reaper-rocky9-testbed}"

docker build -t "${IMAGE_NAME}" -f "${REPO_ROOT}/docker/rocky9/Dockerfile" "${REPO_ROOT}"

docker run --rm \
    --user "$(id -u):$(id -g)" \
    -e HOME=/tmp \
    -e GOCACHE=/tmp/go-build \
    -e GOPATH=/tmp/go \
    -v "${REPO_ROOT}:/workspace" \
    -w /workspace \
    "${IMAGE_NAME}" \
    bash -lc '
set -euo pipefail
rm -rf build dist process-reaper
go version
python3 --version
go vet ./...
go test ./...
go build ./...
bash test/fire_test.sh
mkdir -p build dist
CGO_ENABLED=0 go build -o build/process-reaper ./cmd/process-reaper
nfpm package --config nfpm.yaml --target dist/ --packager rpm
nfpm package --config nfpm.yaml --target dist/ --packager deb
'
