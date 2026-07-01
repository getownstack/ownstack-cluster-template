#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
legacy_script="$repo_root/scripts/setup_system_legacy.sh"
local_binary="$repo_root/bin/ownstackctl"

download_and_run() {
  local url="$1"
  local checksum_url="${2:-}"
  local tmp_dir asset_name tmp_binary

  tmp_dir="$(mktemp -d)"
  asset_name="${url##*/}"
  tmp_binary="$tmp_dir/$asset_name"
  cleanup() { rm -rf "$tmp_dir"; }
  trap cleanup EXIT

  echo "Downloading ownstackctl..."
  curl -fsSL "$url" -o "$tmp_binary"
  chmod +x "$tmp_binary"

  if [ -n "$checksum_url" ]; then
    echo "Verifying ownstackctl checksum..."
    curl -fsSL "$checksum_url" -o "$tmp_dir/checksums.txt"
    (cd "$tmp_dir" && sha256sum -c checksums.txt --ignore-missing)
  fi

  exec "$tmp_binary" apply --legacy-script "$legacy_script"
}

if [ -x "$local_binary" ]; then
  exec "$local_binary" apply --legacy-script "$legacy_script"
fi

if [ -n "${OWNSTACKCTL_URL:-}" ]; then
  download_and_run "$OWNSTACKCTL_URL" "${OWNSTACKCTL_CHECKSUM_URL:-}"
fi

if [ -n "${OWNSTACKCTL_VERSION:-}" ]; then
  case "$(uname -m)" in
    x86_64|amd64) ownstack_arch="amd64" ;;
    aarch64|arm64) ownstack_arch="arm64" ;;
    *) echo "Unsupported architecture for ownstackctl release: $(uname -m)" >&2; exit 1 ;;
  esac

  ownstack_repo="${OWNSTACKCTL_REPO:-getownstack/ownstack-cluster-template}"
  release_base="https://github.com/$ownstack_repo/releases/download/$OWNSTACKCTL_VERSION"
  download_and_run \
    "$release_base/ownstackctl-linux-$ownstack_arch" \
    "$release_base/checksums-linux-$ownstack_arch.txt"
fi

if command -v go >/dev/null 2>&1; then
  exec go run "$repo_root/cmd/ownstackctl" apply --legacy-script "$legacy_script"
fi

echo "ownstackctl binary not found and Go is not installed; falling back to legacy shell installer."
exec bash "$legacy_script"
