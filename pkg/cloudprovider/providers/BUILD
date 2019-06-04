package(default_visibility = ["//visibility:public"])

load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["providers.go"],
    importpath = "k8s.io/kubernetes/pkg/cloudprovider/providers",
    visibility = [
        "//cmd/cloud-controller-manager:__pkg__",
        "//cmd/kube-apiserver/app/options:__pkg__",
        "//cmd/kube-controller-manager/app:__pkg__",
        "//cmd/kubelet/app:__pkg__",
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
