load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = ["builder.go"],
    importmap = "k8s.io/kubernetes/vendor/k8s.io/cli-runtime/pkg/kustomize",
    importpath = "k8s.io/cli-runtime/pkg/kustomize",
    visibility = ["//visibility:public"],
    deps = [
        "//staging/src/k8s.io/cli-runtime/pkg/kustomize/k8sdeps:go_default_library",
        "//vendor/sigs.k8s.io/kustomize/pkg/commands/build:go_default_library",
        "//vendor/sigs.k8s.io/kustomize/pkg/fs:go_default_library",
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
        "//staging/src/k8s.io/cli-runtime/pkg/kustomize/k8sdeps:all-srcs",
    ],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
