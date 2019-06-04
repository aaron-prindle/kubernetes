load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["reconciler.go"],
    importpath = "k8s.io/kubernetes/pkg/kubelet/pluginmanager/reconciler",
    visibility = ["//visibility:public"],
    deps = [
        "//pkg/kubelet/pluginmanager/cache:go_default_library",
        "//pkg/kubelet/pluginmanager/operationexecutor:go_default_library",
        "//pkg/util/goroutinemap:go_default_library",
        "//pkg/util/goroutinemap/exponentialbackoff:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/util/wait:go_default_library",
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
