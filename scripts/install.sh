#!/usr/bin/env bash

set -euo pipefail

readonly repo="ametel01/agents-toolbelt"
readonly binary_name="atb"

version="${ATB_VERSION:-latest}"
install_dir="${ATB_INSTALL_DIR:-/usr/local/bin}"

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "missing required command: $1" >&2
    exit 1
  fi
}

detect_os() {
  case "$(uname -s)" in
    Linux) echo "linux" ;;
    Darwin) echo "darwin" ;;
    *)
      echo "unsupported operating system: $(uname -s)" >&2
      exit 1
      ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *)
      echo "unsupported architecture: $(uname -m)" >&2
      exit 1
      ;;
  esac
}

download_url() {
  local archive="$1"

  if [[ "$version" == "latest" ]]; then
    echo "https://github.com/${repo}/releases/latest/download/${archive}"
    return
  fi

  echo "https://github.com/${repo}/releases/download/${version}/${archive}"
}

install_binary() {
  local source_path="$1"
  local target_path="${install_dir}/${binary_name}"

  mkdir -p "$install_dir" 2>/dev/null || true

  if [[ -w "$install_dir" ]]; then
    install -m 0755 "$source_path" "$target_path"
    return
  fi

  if command -v sudo >/dev/null 2>&1; then
    sudo install -d -m 0755 "$install_dir"
    sudo install -m 0755 "$source_path" "$target_path"
    return
  fi

  echo "install directory is not writable: $install_dir" >&2
  echo "set ATB_INSTALL_DIR to a writable directory on your PATH" >&2
  exit 1
}

main() {
  require_cmd curl
  require_cmd tar
  require_cmd install

  local os
  local arch
  local archive
  local url
  local tmpdir

  os="$(detect_os)"
  arch="$(detect_arch)"
  archive="${binary_name}_${os}_${arch}.tar.gz"
  url="$(download_url "$archive")"
  tmpdir="$(mktemp -d)"
  trap 'rm -rf "$tmpdir"' EXIT

  curl -fsSL "$url" -o "${tmpdir}/${archive}"
  tar -xzf "${tmpdir}/${archive}" -C "$tmpdir"
  install_binary "${tmpdir}/${binary_name}"

  echo "installed ${binary_name} to ${install_dir}/${binary_name}"
  "${install_dir}/${binary_name}" --version
}

main "$@"
