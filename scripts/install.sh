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

verify_checksum() {
  local file="$1"
  local checksums="$2"
  local name="$3"

  local expected
  expected="$(grep "  ${name}$" "$checksums" | head -n1 | cut -d' ' -f1)"

  if [[ -z "$expected" ]]; then
    fail \
      "checksum not found for ${name} in checksums.txt" \
      "the release may be incomplete or the asset name may not match"
  fi

  local actual
  if command -v sha256sum >/dev/null 2>&1; then
    actual="$(sha256sum "$file" | cut -d' ' -f1)"
  elif command -v shasum >/dev/null 2>&1; then
    actual="$(shasum -a 256 "$file" | cut -d' ' -f1)"
  else
    fail \
      "no sha256 utility found" \
      "install sha256sum or shasum and rerun the installer"
  fi

  if [[ "$actual" != "$expected" ]]; then
    fail \
      "checksum verification failed for ${name}" \
      "expected: ${expected}" \
      "actual:   ${actual}" \
      "the download may be corrupted or tampered with"
  fi
}

install_binary() {
  local source_path="$1"
  local target_path="${install_dir}/${binary_name}"

  if ! mkdir -p "$install_dir" 2>&1; then
    fail \
      "failed to create install directory: $install_dir" \
      "check that the parent directory exists and you have write permission" \
      "set ATB_INSTALL_DIR to a writable directory on your PATH"
  fi

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

  local checksums_url
  checksums_url="$(download_url "checksums.txt")"

  if ! curl -fsSL "$checksums_url" -o "${tmpdir}/checksums.txt"; then
    fail \
      "failed to download checksums: $checksums_url" \
      "check your network connection and confirm the requested release exists" \
      "if you set ATB_VERSION, verify that the tag is published in GitHub releases"
  fi

  if ! curl -fsSL "$url" -o "${tmpdir}/${archive}"; then
    fail \
      "failed to download release archive: $url" \
      "check your network connection and confirm the requested release exists" \
      "if you set ATB_VERSION, verify that the tag is published in GitHub releases"
  fi

  verify_checksum "${tmpdir}/${archive}" "${tmpdir}/checksums.txt" "$archive"

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
