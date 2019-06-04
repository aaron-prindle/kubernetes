load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "cpu_assignment.go",
        "cpu_manager.go",
        "fake_cpu_manager.go",
        "policy.go",
        "policy_none.go",
        "policy_static.go",
    ],
    importpath = "k8s.io/kubernetes/pkg/kubelet/cm/cpumanager",
    visibility = ["//visibility:public"],
    deps = [
        "//pkg/apis/core/v1/helper/qos:go_default_library",
        "//pkg/kubelet/cm/cpumanager/state:go_default_library",
        "//pkg/kubelet/cm/cpumanager/topology:go_default_library",
        "//pkg/kubelet/cm/cpuset:go_default_library",
        "//pkg/kubelet/container:go_default_library",
        "//pkg/kubelet/status:go_default_library",
        "//staging/src/k8s.io/api/core/v1:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/util/wait:go_default_library",
        "//staging/src/k8s.io/cri-api/pkg/apis/runtime/v1alpha2:go_default_library",
        "//vendor/github.com/google/cadvisor/info/v1:go_default_library",
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
        "//pkg/kubelet/cm/cpumanager/state:all-srcs",
        "//pkg/kubelet/cm/cpumanager/topology:all-srcs",
    ],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
