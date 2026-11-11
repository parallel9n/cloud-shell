#!/usr/bin/env bash
# Build static cloud-shell archives for macOS, Linux, and Windows.
set -euo pipefail

version="${1:-dev}"
out="${2:-dist}"
name_version="${version#v}"

mkdir -p "$out"
rm -f "$out"/cloud-shell_* "$out"/SHA256SUMS

targets=(
  darwin/amd64
  darwin/arm64
  linux/amd64
  linux/arm64
  windows/amd64
  windows/arm64
)

for target in "${targets[@]}"; do
  goos="${target%/*}"
  goarch="${target#*/}"
  bin="cloud-shell"
  if [[ "$goos" == "windows" ]]; then
    bin="cloud-shell.exe"
  fi
  echo "building ${goos}/${goarch}"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
    go build -trimpath -ldflags="-s -w -X main.version=${version}" -o "$out/$bin" .
  archive="cloud-shell_${name_version}_${goos}_${goarch}"
  if [[ "$goos" == "windows" ]]; then
    (cd "$out" && zip -q "${archive}.zip" "$bin")
  else
    tar -C "$out" -czf "$out/${archive}.tar.gz" "$bin"
  fi
  rm -f "$out/$bin"
done

(
  cd "$out"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum cloud-shell_* > SHA256SUMS
  else
    shasum -a 256 cloud-shell_* > SHA256SUMS
  fi
)
