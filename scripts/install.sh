#!/usr/bin/env bash

set -euo pipefail

readonly repo="ametel01/agents-toolbelt"
readonly binary_name="atb"

version="${ATB_VERSION:-latest}"
install_dir=""
tmpdir=""

fail() {
  echo "$1" >&2
  shift

  while [[ "$#" -gt 0 ]]; do
    echo "resolution: $1" >&2
    shift
  done

  exit 1
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    case "$1" in
      curl)
        fail \
          "missing required command: curl" \
          "install curl with your system package manager and rerun the installer"
        ;;
      tar)
        fail \
          "missing required command: tar" \
          "install tar with your system package manager and rerun the installer"
        ;;
      install)
        fail \
          "missing required command: install" \
          "install GNU coreutils or the equivalent base system utilities for your platform"
        ;;
      *)
        fail \
          "missing required command: $1" \
          "install $1 and rerun the installer"
        ;;
    esac
  fi
}

detect_os() {
  case "$(uname -s)" in
    Linux) echo "linux" ;;
    Darwin) echo "darwin" ;;
    *)
      fail \
        "unsupported operating system: $(uname -s)" \
        "supported operating systems are Linux and macOS"
      ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *)
      fail \
        "unsupported architecture: $(uname -m)" \
        "supported architectures are amd64 and arm64"
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

resolve_install_dir() {
  if [[ -n "${ATB_INSTALL_DIR:-}" ]]; then
    echo "$ATB_INSTALL_DIR"
    return
  fi

  if [[ "${EUID:-$(id -u)}" -eq 0 ]]; then
    echo "/usr/local/bin"
    return
  fi

  echo "${HOME}/.local/bin"
}

install_binary() {
  local source_path="$1"
  local target_path="${install_dir}/${binary_name}"

  mkdir -p "$install_dir" 2>/dev/null || true

  if [[ -w "$install_dir" ]]; then
    install -m 0755 "$source_path" "$target_path"
    return
  fi

  fail \
    "install directory is not writable: $install_dir" \
    "set ATB_INSTALL_DIR to a writable directory on your PATH, for example \$HOME/.local/bin" \
    "for a system-wide install, inspect the script first and then run it explicitly with sudo" \
    "the installer will not invoke sudo automatically"
}

cleanup() {
  if [[ -n "$tmpdir" ]]; then
    rm -rf "$tmpdir"
  fi
}

main() {
  require_cmd curl
  require_cmd tar
  require_cmd install

  local os
  local arch
  local archive
  local url

  install_dir="$(resolve_install_dir)"
  os="$(detect_os)"
  arch="$(detect_arch)"
  archive="${binary_name}_${os}_${arch}.tar.gz"
  url="$(download_url "$archive")"
  tmpdir="$(mktemp -d)"
  trap cleanup EXIT

  if ! curl -fsSL "$url" -o "${tmpdir}/${archive}"; then
    fail \
      "failed to download release archive: $url" \
      "check your network connection and confirm the requested release exists" \
      "if you set ATB_VERSION, verify that the tag is published in GitHub releases"
  fi

  if ! tar -xzf "${tmpdir}/${archive}" -C "$tmpdir"; then
    fail \
      "failed to extract release archive: ${tmpdir}/${archive}" \
      "the download may be incomplete or the release asset may not match this platform"
  fi

  if [[ ! -f "${tmpdir}/${binary_name}" ]]; then
    fail \
      "release archive did not contain expected binary: ${binary_name}" \
      "check that the published release asset was built correctly for ${os}/${arch}"
  fi

  install_binary "${tmpdir}/${binary_name}"

  echo "installed ${binary_name} to ${install_dir}/${binary_name}"
  "${install_dir}/${binary_name}" --version
}

main "$@"
