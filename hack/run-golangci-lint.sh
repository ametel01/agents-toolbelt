#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
config_path="${repo_root}/.golangci.yml"
golangci_bin="${1}"

shift

if "${golangci_bin}" help linters | grep -q '^gosimple:'; then
	exec "${golangci_bin}" run -c "${config_path}" "$@"
fi

tmp_config="$(mktemp "${TMPDIR:-/tmp}/golangci.XXXXXX.yml")"
trap 'rm -f "${tmp_config}"' EXIT

sed '/^[[:space:]]*- gosimple$/d' "${config_path}" > "${tmp_config}"

exec "${golangci_bin}" run -c "${tmp_config}" "$@"
