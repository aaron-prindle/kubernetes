load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "interfaces.go",
        "legacy_metrics_client.go",
        "rest_metrics_client.go",
        "utilization.go",
    ],
    importpath = "k8s.io/kubernetes/pkg/controller/podautoscaler/metrics",
    visibility = ["//visibility:public"],
    deps = [
        "//staging/src/k8s.io/api/autoscaling/v2beta2:go_default_library",
        "//staging/src/k8s.io/api/core/v1:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/apis/meta/v1:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/labels:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/runtime/schema:go_default_library",
        "//staging/src/k8s.io/client-go/kubernetes:go_default_library",
        "//staging/src/k8s.io/client-go/kubernetes/typed/core/v1:go_default_library",
        "//staging/src/k8s.io/metrics/pkg/apis/custom_metrics/v1beta2:go_default_library",
        "//staging/src/k8s.io/metrics/pkg/apis/metrics/v1alpha1:go_default_library",
        "//staging/src/k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1:go_default_library",
        "//staging/src/k8s.io/metrics/pkg/client/custom_metrics:go_default_library",
        "//staging/src/k8s.io/metrics/pkg/client/external_metrics:go_default_library",
        "//vendor/k8s.io/heapster/metrics/api/v1/types:go_default_library",
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
