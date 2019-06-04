package(default_visibility = ["//visibility:public"])

load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "namespace.go",
        "runtime.go",
        "whitelist.go",
    ],
    importpath = "k8s.io/kubernetes/pkg/kubelet/sysctl",
    deps = [
        "//pkg/apis/core/validation:go_default_library",
        "//pkg/apis/policy/validation:go_default_library",
        "//pkg/kubelet/container:go_default_library",
        "//pkg/kubelet/lifecycle:go_default_library",
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
    srcs = [":package-srcs"],
    tags = ["automanaged"],
)
