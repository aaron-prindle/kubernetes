package(default_visibility = ["//visibility:public"])

load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "allocator.go",
        "operation.go",
    ],
    importpath = "k8s.io/kubernetes/pkg/registry/core/service/portallocator",
    deps = [
        "//pkg/apis/core:go_default_library",
        "//pkg/registry/core/service/allocator:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/util/net:go_default_library",
        "//vendor/k8s.io/klog:go_default_library",
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
        "//pkg/registry/core/service/portallocator/controller:all-srcs",
    ],
    tags = ["automanaged"],
)
