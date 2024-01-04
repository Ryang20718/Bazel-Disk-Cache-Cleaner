load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library")

go_library(
    name = "bazel_disk_cache_cleaner_lib",
    srcs = ["bazel_disk_cache_cleaner.go"],
    importpath = "github.com/Ryang20718/bazel-disk-cache-cleaner/bazel_disk_cache_cleaner",
    visibility = ["//visibility:private"],
    deps = [
        "@com_github_spf13_cobra//:cobra",
        "@org_uber_go_zap//:go_default_library",
    ],
)

go_binary(
    name = "bazel_disk_cache_cleaner",
    embed = [":bazel_disk_cache_cleaner_lib"],
    visibility = ["//visibility:public"],
)
