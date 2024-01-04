#!/bin/bash

set -euo pipefail

# Fetch all external repos still in use
# Reason for doing this is bazel doesn't update atime on external repos; Bazel keeps these repos in memory.
# However, data artifacts may remain in this directory over time and thus fill the cache
# We query for these repos and only clear from this directory for any repos that aren't found within the bazel query
USERNAME=$(/usr/bin/logname)

mkdir -p "/tmp/${USERNAME}"
external_repo_list="/tmp/${USERNAME}/bazel_external_repos.txt"

if [[ -z ${BAZEL_CACHE_DIR+x} ]]; then
    BAZEL_CACHE_DIR="$(readlink -f /home/${USERNAME}/.cache/bazel/_bazel_${USERNAME})"
fi

if [[ -z ${KEEP_FILES_ACCESSED_DAYS+x} ]]; then
    KEEP_FILES_ACCESSED_DAYS=5
fi

SET_VERBOSE=""
if [[ -n ${VERBOSE+x} ]]; then
    SET_VERBOSE="--verbose"
fi

# raw data is //external:<target>. strip away //external: for easier processing from Go's side
tools/bazel query //external:* | sed 's|//external:||' >"${external_repo_list}"

# This script concurrently with other bazel processes. Thus, we must hold the lock
echo "Clean Bazel Cache script is now holding the bazel lock, preventing other bazel processes from running"
echo "This script is cleaning out unused files from the bazel cache so you don't need to expunge!"
bazel_locks_path="${BAZEL_CACHE_DIR}"
for file in "${bazel_locks_path}"/*/lock; do
    sudo sed -i "s/pid=[0-9]*/pid=$$/g" "$file"
done

before_storage_used="$(du -sh ${BAZEL_CACHE_DIR} &)"
clean_bazel_cache_binary=bazel-out/k8-opt/bin/tools/clean_bazel_cache/clean_bazel_cache_/clean_bazel_cache
if [[ ! -f $clean_bazel_cache_binary ]]; then
    # Build binary if this tool doesn't exist
    tools/bazel build //tools/clean_bazel_cache
fi

# bazel cache requires sudo to remove
sudo ${clean_bazel_cache_binary} clean \
    "${SET_VERBOSE}" \
    --bazel-cache-dir "${BAZEL_CACHE_DIR}" \
    --keep-files-access-days "${KEEP_FILES_ACCESSED_DAYS}" \
    --external-repo-target-list "${external_repo_list}"

# Statistics for storage cleared
after_storage_used="$(du -sh ${BAZEL_CACHE_DIR} &)"
echo "Bazel Cache Disk Usage before cleaning: ${before_storage_used}"
echo "Bazel Cache Disk Usage after cleaning: ${after_storage_used}"

# Reset Bazel Memory state now that we've cleaned the cache
tools/bazel shutdown
