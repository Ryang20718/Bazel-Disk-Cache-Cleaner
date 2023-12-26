#!/bin/bash

set -euo pipefail

USERNAME=$(/usr/bin/logname)
if [[ -z "${BAZEL_CACHE_DIR+x}" ]]; then
    BAZEL_CACHE_DIR="$(readlink -f /home/$USERNAME/.cache/bazel/_bazel_$USERNAME)"
fi

if [[ -z "${KEEP_FILES_ACCESSED_DAYS+x}" ]]; then
    KEEP_FILES_ACCESSED_DAYS=5
fi

before_storage_used=$(du -sh ${BAZEL_CACHE_DIR} &)
bazel_cache_binary=bazel-out/k8-opt/bin/tools/bazel_disk_cache_cleaner/bazel_disk_cache_cleaner_/bazel_disk_cache_cleaner
if [[ ! -f "$bazel_cache_binary" ]]; then
    # Build binary if this tool doesn't exist
    tools/bazel build //tools/bazel_disk_cache_cleaner
fi

# bazel cache requires sudo to remove
sudo ${bazel_cache_binary} clean \
    --bazel-cache-dir ${BAZEL_CACHE_DIR} \
    --keep-files-access-days ${KEEP_FILES_ACCESSED_DAYS}

# Statistics for storage cleared
after_storage_used=$(du -sh ${BAZEL_CACHE_DIR} &)
echo "Bazel Cache Disk Usage before cleaning: ${before_storage_used}"
echo "Bazel Cache Disk Usage after cleaning: ${after_storage_used}"
