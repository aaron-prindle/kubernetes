load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "comparer.go",
        "debugger.go",
        "dumper.go",
        "signal.go",
        "signal_windows.go",
    ],
    importpath = "k8s.io/kubernetes/pkg/scheduler/internal/cache/debugger",
    visibility = ["//pkg/scheduler:__subpackages__"],
    deps = [
        "//pkg/scheduler/internal/cache:go_default_library",
        "//pkg/scheduler/internal/queue:go_default_library",
        "//pkg/scheduler/nodeinfo:go_default_library",
        "//staging/src/k8s.io/api/core/v1:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/labels:go_default_library",
        "//staging/src/k8s.io/client-go/listers/core/v1:go_default_library",
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
    srcs = [":package-srcs"],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
