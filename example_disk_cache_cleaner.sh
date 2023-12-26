#!/bin/bash

# Wrapper script for bazel_disk_cache_cleaner
set -eu -o pipefail

platform="$(uname -s | tr '[:upper:]' '[:lower:]')"
os=${OSTYPE%%-*}
os=${os//[0-9]/}
arch="$(uname -m)"

VERSION=v0.0.4
INSTALL_ROOT="${HOME}/.cache/bazel_disk_cache_cleaner/${VERSION}"
URL=https://github.com/Ryang20718/Bazel-Disk-Cache-Cleaner/releases/download/${VERSION}/bazel_disk_cache_cleaner_${os}_${arch}
mkdir -p "${INSTALL_ROOT}"
if [[ ! -f "${INSTALL_ROOT}/${VERSION}/bazel_disk_cache_cleaner_${os}_${arch}" ]]; then
    wget -q ${URL} -P "${INSTALL_ROOT}"
fi

TOOL_PATH="${INSTALL_ROOT}/${VERSION}/bazel_disk_cache_cleaner_${os}_${arch}"

exec ${TOOL_PATH} "$@"