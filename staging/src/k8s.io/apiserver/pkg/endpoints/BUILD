package(default_visibility = ["//visibility:public"])

load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "go_default_library",
    srcs = [
        "doc.go",
        "groupversion.go",
        "installer.go",
    ],
    importmap = "k8s.io/kubernetes/vendor/k8s.io/apiserver/pkg/endpoints",
    importpath = "k8s.io/apiserver/pkg/endpoints",
    deps = [
        "//staging/src/k8s.io/apimachinery/pkg/apis/meta/v1:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/conversion:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/runtime:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/runtime/schema:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/types:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/util/errors:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/util/sets:go_default_library",
        "//staging/src/k8s.io/apiserver/pkg/admission:go_default_library",
        "//staging/src/k8s.io/apiserver/pkg/authorization/authorizer:go_default_library",
        "//staging/src/k8s.io/apiserver/pkg/endpoints/discovery:go_default_library",
        "//staging/src/k8s.io/apiserver/pkg/endpoints/handlers:go_default_library",
        "//staging/src/k8s.io/apiserver/pkg/endpoints/handlers/fieldmanager:go_default_library",
        "//staging/src/k8s.io/apiserver/pkg/endpoints/handlers/negotiation:go_default_library",
        "//staging/src/k8s.io/apiserver/pkg/endpoints/metrics:go_default_library",
        "//staging/src/k8s.io/apiserver/pkg/features:go_default_library",
        "//staging/src/k8s.io/apiserver/pkg/registry/rest:go_default_library",
        "//staging/src/k8s.io/apiserver/pkg/server/filters:go_default_library",
        "//staging/src/k8s.io/apiserver/pkg/util/feature:go_default_library",
        "//vendor/github.com/emicklei/go-restful:go_default_library",
        "//vendor/k8s.io/kube-openapi/pkg/util/proto:go_default_library",
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
        "//staging/src/k8s.io/apiserver/pkg/endpoints/discovery:all-srcs",
        "//staging/src/k8s.io/apiserver/pkg/endpoints/filters:all-srcs",
        "//staging/src/k8s.io/apiserver/pkg/endpoints/handlers:all-srcs",
        "//staging/src/k8s.io/apiserver/pkg/endpoints/metrics:all-srcs",
        "//staging/src/k8s.io/apiserver/pkg/endpoints/openapi:all-srcs",
        "//staging/src/k8s.io/apiserver/pkg/endpoints/request:all-srcs",
        "//staging/src/k8s.io/apiserver/pkg/endpoints/testing:all-srcs",
    ],
    tags = ["automanaged"],
)
