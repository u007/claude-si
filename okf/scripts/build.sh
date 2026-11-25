#!/bin/sh
set -eu

cd "$(dirname -- "$0")/.."
export CGO_ENABLED=0
export GOTOOLCHAIN=go1.24.0

version=$(go env GOVERSION)
if [ "$version" != "go1.24.0" ]; then
  echo "okf: build requires go1.24.0, found $version" >&2
  exit 1
fi

for target in darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64 windows/arm64; do
  os=${target%/*}
  arch=${target#*/}
  case "$os" in
    windows) suffix=.exe ;;
    *) suffix= ;;
  esac
  mkdir -p "bin/$os/$arch"
  GOOS="$os" GOARCH="$arch" go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o "bin/$os/$arch/okf$suffix" ./cmd/okf
done
