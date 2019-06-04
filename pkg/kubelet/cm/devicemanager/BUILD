load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "device_plugin_stub.go",
        "endpoint.go",
        "manager.go",
        "manager_stub.go",
        "pod_devices.go",
        "types.go",
    ],
    importpath = "k8s.io/kubernetes/pkg/kubelet/cm/devicemanager",
    visibility = ["//visibility:public"],
    deps = [
        "//pkg/apis/core/v1/helper:go_default_library",
        "//pkg/features:go_default_library",
        "//pkg/kubelet/apis/deviceplugin/v1beta1:go_default_library",
        "//pkg/kubelet/apis/pluginregistration/v1:go_default_library",
        "//pkg/kubelet/apis/podresources/v1alpha1:go_default_library",
        "//pkg/kubelet/checkpointmanager:go_default_library",
        "//pkg/kubelet/checkpointmanager/errors:go_default_library",
        "//pkg/kubelet/cm/devicemanager/checkpoint:go_default_library",
        "//pkg/kubelet/config:go_default_library",
        "//pkg/kubelet/container:go_default_library",
        "//pkg/kubelet/lifecycle:go_default_library",
        "//pkg/kubelet/metrics:go_default_library",
        "//pkg/kubelet/pluginmanager/cache:go_default_library",
        "//pkg/scheduler/nodeinfo:go_default_library",
        "//pkg/util/selinux:go_default_library",
        "//staging/src/k8s.io/api/core/v1:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/api/resource:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/util/sets:go_default_library",
        "//staging/src/k8s.io/apiserver/pkg/util/feature:go_default_library",
        "//vendor/google.golang.org/grpc:go_default_library",
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
        "//pkg/kubelet/cm/devicemanager/checkpoint:all-srcs",
    ],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)
