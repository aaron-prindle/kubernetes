load("@io_k8s_repo_infra//defs:pkg.bzl", "pkg_tar")
load("@io_k8s_repo_infra//defs:build.bzl", "release_filegroup")

# Having the COS code from the GCE cluster deploy hosted with the release is
# useful for GKE.  This list should match the list in
# kubernetes/release/lib/releaselib.sh.
release_filegroup(
    name = "gcs-release-artifacts",
    srcs = [
        "configure.sh",
        "master.yaml",
        "node.yaml",
        "shutdown.sh",
    ],
    visibility = ["//visibility:public"],
)

pkg_tar(
    name = "gci-trusty-manifests",
    srcs = glob(["gke-internal-configure-helper.sh"]),
    files = {
        "//cluster/gce/gci/mounter": "gci-mounter",
        "configure-helper.sh": "gci-configure-helper.sh",
        "health-monitor.sh": "health-monitor.sh",
    },
    mode = "0755",
    strip_prefix = ".",
    visibility = ["//visibility:public"],
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
        "//cluster/gce/gci/mounter:all-srcs",
    ],
    tags = ["automanaged"],
    visibility = ["//visibility:public"],
)

filegroup(
    name = "scripts-test-data",
    srcs = [
        "configure-helper.sh",
    ],
)
