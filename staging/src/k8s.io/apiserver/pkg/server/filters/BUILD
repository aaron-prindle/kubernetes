package(default_visibility = ["//visibility:public"])

load(
    "@io_bazel_rules_go//go:def.bzl",
    "go_library",
    "go_test",
)

go_test(
    name = "go_default_test",
    srcs = [
        "content_type_test.go",
        "cors_test.go",
        "maxinflight_test.go",
        "reqmgmt_test.go",
        "timeout_test.go",
    ],
    embed = [":go_default_library"],
    deps = [
        "//staging/src/k8s.io/api/flowcontrol/v1alpha1:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/api/errors:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/apis/meta/v1:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/runtime/schema:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/util/clock:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/util/diff:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/util/runtime:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/util/sets:go_default_library",
        "//staging/src/k8s.io/apiserver/pkg/authentication/user:go_default_library",
        "//staging/src/k8s.io/apiserver/pkg/endpoints/filters:go_default_library",
        "//staging/src/k8s.io/apiserver/pkg/endpoints/request:go_default_library",
        "//staging/src/k8s.io/client-go/kubernetes:go_default_library",
        "//staging/src/k8s.io/client-go/kubernetes/fake:go_default_library",
    ],
)

go_library(
    name = "go_default_library",
    srcs = [
        "content_type.go",
        "cors.go",
        "doc.go",
        "longrunning.go",
        "maxinflight.go",
        "reqmgmt.go",
        "reqmgmt-config.go",
        "requestor.go",
        "timeout.go",
        "waitgroup.go",
        "wrap.go",
    ],
    importmap = "k8s.io/kubernetes/vendor/k8s.io/apiserver/pkg/server/filters",
    importpath = "k8s.io/apiserver/pkg/server/filters",
    deps = [
        "//staging/src/k8s.io/apimachinery/pkg/api/errors:go_default_library",
<<<<<<< HEAD
=======
        "//staging/src/k8s.io/apimachinery/pkg/labels:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/util/clock:go_default_library",
>>>>>>> Initial logic for consuming config objects in prioritization and faireness filter
        "//staging/src/k8s.io/apimachinery/pkg/util/runtime:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/util/sets:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/util/wait:go_default_library",
        "//staging/src/k8s.io/apimachinery/pkg/util/waitgroup:go_default_library",
        "//staging/src/k8s.io/apiserver/pkg/authentication/user:go_default_library",
        "//staging/src/k8s.io/apiserver/pkg/endpoints/handlers/responsewriters:go_default_library",
        "//staging/src/k8s.io/apiserver/pkg/endpoints/metrics:go_default_library",
        "//staging/src/k8s.io/apiserver/pkg/endpoints/request:go_default_library",
        "//staging/src/k8s.io/apiserver/pkg/server/httplog:go_default_library",
<<<<<<< HEAD
        "//staging/src/k8s.io/apiserver/pkg/util/flowcontrol:go_default_library",
=======
        "//staging/src/k8s.io/apiserver/pkg/util/flowcontrol/fairqueuing:go_default_library",
        "//staging/src/k8s.io/client-go/informers:go_default_library",
        "//staging/src/k8s.io/client-go/kubernetes:go_default_library",
        "//staging/src/k8s.io/client-go/listers/flowcontrol/v1alpha1:go_default_library",
        "//staging/src/k8s.io/client-go/rest:go_default_library",
        "//staging/src/k8s.io/client-go/tools/cache:go_default_library",
        "//staging/src/k8s.io/client-go/util/workqueue:go_default_library",
        "//vendor/golang.org/x/net/http2:go_default_library",
>>>>>>> SPLIT-OUT: integrate fair queuing into test skeleton, mock out flowschema logic, add WIP passing test suite
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
)
