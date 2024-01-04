load("@bazel_gazelle//:def.bzl", "gazelle")

# gazelle:prefix github.com/Ryang20718/bazel-disk-cache-cleaner
gazelle(name = "gazelle")

gazelle(
    name = "gazelle-update-repos",
    args = [
        "-from_file=go.mod",
        "-to_macro=third_party/deps.bzl%go_dependencies",
        "-prune",
    ],
    command = "update-repos",
)
