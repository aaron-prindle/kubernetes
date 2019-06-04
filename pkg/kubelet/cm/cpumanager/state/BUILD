load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "checkpoint.go",
        "state.go",
        "state_checkpoint.go",
        "state_file.go",
        "state_mem.go",
    ],
    importpath = "k8s.io/kubernetes/pkg/kubelet/cm/cpumanager/state",
    visibility = ["//visibility:public"],
    deps = [
        "//pkg/kubelet/checkpointmanager:go_default_library",
        "//pkg/kubelet/checkpointmanager/checksum:go_default_library",
        "//pkg/kubelet/checkpointmanager/errors:go_default_library",
        "//pkg/kubelet/cm/cpuset:go_default_library",
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
        "//pkg/kubelet/cm/cpumanager/state/testing:all-srcs",
    ],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
