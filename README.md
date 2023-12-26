# Bazel-Disk-Cache-Cleaner

**What is this project for?**
Bazel users on local executors. Bazel eats up disk space and this script is intended to clean your disk space but only files that haven't been accesssed in a while via access time.

## How to use

Leverage the shell script wrapper `bazel_disk_cache_cleaner/bazel_disk_cache_cleaner.sh` to clean your bazel cache!
Replace `<path-to-bazel-cache-directory>` with the path to your Bazel cache directory and `<number-of-days>` with the number of days after which you want to delete files. 

## Flags

- `--bazel-cache-dir`: Path to the Bazel cache directory to clear.
- `--keep-files-access-days`: Purge files with access time greater than the specified number of days.

## Commands

- `clean`: Cleans the Bazel cache.

## Functionality

This CLI iterates through the entire `BAZEL_CACHE_DIR` env variable set and deletes all files older than `KEEP_FILES_ACCESSED_DAYS`

## Prequisites

1. Install Bazel (Prefer bazelisk)

**To run formatters**
`tools/trunk fmt`

**To do all your GO development & bazel development**
`tools/trunk check` has got you covered with vendoring third party dependencies and generating BUILD files. See `.trunk.yaml` for custom definitions. Also, linters are run here as well :)
