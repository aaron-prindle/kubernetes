package(default_visibility = ["//visibility:public"])

load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["checkpoint_manager.go"],
    importpath = "k8s.io/kubernetes/pkg/kubelet/checkpointmanager",
    deps = [
        "//pkg/kubelet/checkpointmanager/errors:go_default_library",
        "//pkg/kubelet/util/store:go_default_library",
        "//pkg/util/filesystem:go_default_library",
    ],
)

filegroup(
    name = "package-srcs",
    srcs = glob(["**"]),
    tags = ["automanaged"],
    visibility = ["//visibility:private"],
)

filegroup(
    name = "all-srcs",
    srcs = [
        ":package-srcs",
        "//pkg/kubelet/checkpointmanager/checksum:all-srcs",
        "//pkg/kubelet/checkpointmanager/errors:all-srcs",
        "//pkg/kubelet/checkpointmanager/testing:all-srcs",
    ],
    tags = ["automanaged"],
)
