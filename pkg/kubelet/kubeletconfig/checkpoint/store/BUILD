package(default_visibility = ["//visibility:public"])

load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "fakestore.go",
        "fsstore.go",
        "store.go",
    ],
    importpath = "k8s.io/kubernetes/pkg/kubelet/kubeletconfig/checkpoint/store",
    deps = [
        "//pkg/kubelet/apis/config:go_default_library",
        "//pkg/kubelet/kubeletconfig/checkpoint:go_default_library",
        "//pkg/kubelet/kubeletconfig/configfiles:go_default_library",
        "//pkg/kubelet/kubeletconfig/util/files:go_default_library",
        "//pkg/kubelet/kubeletconfig/util/log:go_default_library",
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
    srcs = [":package-srcs"],
    tags = ["automanaged"],
)
