package(default_visibility = ["//visibility:public"])

load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "doc.go",
        "fake_shaper.go",
        "interfaces.go",
        "linux.go",
        "unsupported.go",
        "utils.go",
    ],
    importpath = "k8s.io/kubernetes/pkg/util/bandwidth",
    deps = [
        "//staging/src/k8s.io/apimachinery/pkg/api/resource:go_default_library",
    ] + select({
        "@io_bazel_rules_go//go/platform:linux": [
            "//staging/src/k8s.io/apimachinery/pkg/util/sets:go_default_library",
            "//vendor/k8s.io/klog:go_default_library",
            "//vendor/k8s.io/utils/exec:go_default_library",
        ],
        "//conditions:default": [],
    }),
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
