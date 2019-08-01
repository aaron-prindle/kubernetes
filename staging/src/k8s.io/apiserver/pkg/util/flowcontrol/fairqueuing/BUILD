<<<<<<< HEAD
load("@io_bazel_rules_go//go:def.bzl", "go_library")
=======
load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")
>>>>>>> add fair queuing implementation and passing test suite

go_library(
    name = "go_default_library",
    srcs = [
<<<<<<< HEAD
        "interface.go",
        "no-restraint.go",
=======
        "fairqueuing.go",
        "types.go",
>>>>>>> add fair queuing implementation and passing test suite
    ],
    importmap = "k8s.io/kubernetes/vendor/k8s.io/apiserver/pkg/util/flowcontrol/fairqueuing",
    importpath = "k8s.io/apiserver/pkg/util/flowcontrol/fairqueuing",
    visibility = ["//visibility:public"],
<<<<<<< HEAD
=======
    deps = ["//staging/src/k8s.io/apimachinery/pkg/util/clock:go_default_library"],
)

go_test(
    name = "go_default_test",
    srcs = ["fairqueuing_test.go"],
    embed = [":go_default_library"],
    deps = ["//staging/src/k8s.io/apimachinery/pkg/util/clock:go_default_library"],
>>>>>>> add fair queuing implementation and passing test suite
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
