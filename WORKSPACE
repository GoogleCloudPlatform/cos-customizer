# Copyright 2020 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the License);
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an AS IS BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")
load("@bazel_tools//tools/build_defs/repo:git.bzl", "git_repository")

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "355d40d12749d843cfd05e14c304ac053ae82be4cd257efaf5ef8ce2caf31f1c",
    strip_prefix = "rules_go-197699822e081dad064835a09825448a3e4cc2a2",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/archive/197699822e081dad064835a09825448a3e4cc2a2.tar.gz",
        "https://github.com/bazelbuild/rules_go/archive/197699822e081dad064835a09825448a3e4cc2a2.tar.gz",
    ],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "222e49f034ca7a1d1231422cdb67066b885819885c356673cb1f72f748a3c9d4",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.22.3/bazel-gazelle-v0.22.3.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.22.3/bazel-gazelle-v0.22.3.tar.gz",
    ],
)

http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "4521794f0fba2e20f3bf15846ab5e01d5332e587e9ce81629c7f96c793bb7036",
    strip_prefix = "rules_docker-0.14.4",
    urls = ["https://github.com/bazelbuild/rules_docker/archive/v0.14.4.tar.gz"],
)

http_archive(
    name = "distroless",
    sha256 = "14834aaf9e005b9175de2cfa2b420c80778880ee4d9f9a9f7f385d3b177abff7",
    strip_prefix = "distroless-fa0765cc86064801e42a3b35f50ff2242aca9998",
    urls = ["https://github.com/GoogleContainerTools/distroless/archive/fa0765cc86064801e42a3b35f50ff2242aca9998.tar.gz"],
)

http_archive(
    name = "rules_pkg",
    sha256 = "aeca78988341a2ee1ba097641056d168320ecc51372ef7ff8e64b139516a4937",
    urls = ["https://github.com/bazelbuild/rules_pkg/releases/download/0.2.6-1/rules_pkg-0.2.6.tar.gz"],
)

http_archive(
    name = "rules_foreign_cc",
    sha256 = "ab805b9e00747ba9b184790cbe2d4d19b672770fcac437f01d8c101ae60df996",
    strip_prefix = "rules_foreign_cc-c309ec13192f69a46aaaba39587c3d7ff684eb35",
    urls = ["https://github.com/bazelbuild/rules_foreign_cc/archive/c309ec13192f69a46aaaba39587c3d7ff684eb35.zip"],
)

git_repository(
    name = "com_google_protobuf",
    commit = "31ebe2ac71400344a5db91ffc13c4ddfb7589f92",
    remote = "https://github.com/protocolbuffers/protobuf",
    shallow_since = "1591135967 -0700",
)

git_repository(
    name = "com_github_googlecloudplatform_docker_credential_gcr",
    commit = "6093d30b51d725877bc6971aa6700153c1a364f1",
    remote = "https://github.com/GoogleCloudPlatform/docker-credential-gcr",
    shallow_since = "1613169008 -0800",
)

load("@com_google_protobuf//:protobuf_deps.bzl", "protobuf_deps")

protobuf_deps()

load("@rules_pkg//:deps.bzl", "rules_pkg_dependencies")
load("@io_bazel_rules_go//go:deps.bzl", "go_rules_dependencies", "go_register_toolchains")

go_rules_dependencies()

go_register_toolchains(version="1.16")

load("//:deps.bzl", "go_mod_deps")

# gazelle:repository_macro deps.bzl%go_mod_deps
go_mod_deps()

rules_pkg_dependencies()

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")

gazelle_dependencies()

load(
    "@io_bazel_rules_docker//repositories:repositories.bzl",
    container_repositories = "repositories",
)

container_repositories()

load("@io_bazel_rules_docker//repositories:deps.bzl", container_deps = "deps")

container_deps()

load("@io_bazel_rules_docker//repositories:pip_repositories.bzl", "pip_deps")

pip_deps()

load(
    "@io_bazel_rules_docker//container:container.bzl",
    "container_pull",
)

container_pull(
    name = "daisy",
    digest = "sha256:a23774074d5941ed9e25f64ee7e02f96d2f8e09a4d7cee7131b49664267c33c7",
    registry = "gcr.io",
    repository = "compute-image-tools/daisy",
)

load(
    "@io_bazel_rules_docker//go:image.bzl",
    _go_image_repos = "repositories",
)

_go_image_repos()

load(
    "@distroless//package_manager:package_manager.bzl",
    "package_manager_repositories",
)

package_manager_repositories()

load(
    "@distroless//package_manager:dpkg.bzl",
    "dpkg_src",
    "dpkg_list",
)

dpkg_src(
    name = "debian_stretch",
    arch = "amd64",
    distro = "stretch",
    sha256 = "79a66cd92ba9096fce679e15d0b5feb9effcf618b0a6d065eb32684dbffd0311",
    snapshot = "20190328T105444Z",
    url = "http://snapshot.debian.org/archive",
)

dpkg_list(
    name = "package_bundle",
    packages = [
        "coreutils",
        "libacl1",
        "libattr1",
        "libc6",
        "libpcre3",
        "libselinux1",
        "tar",
        "cryptsetup-bin",
        "libcryptsetup4",
        "libpopt0",
        "libuuid1",
        "libdevmapper1.02.1",
        "libgcrypt20",
        "libargon2-0",
        "libjson-c3",
        "libudev1",
        "libpthread-stubs0-dev",
        "libm17n-0",
        "libgpg-error0",
        "mtools",
    ],
    sources = [
        "@debian_stretch//file:Packages.json",
    ],
)

load("@rules_foreign_cc//:workspace_definitions.bzl", "rules_foreign_cc_dependencies")
rules_foreign_cc_dependencies()

load("//src/third_party/dosfstools:dosfstools_repositories.bzl", "dosfstools_repositories")
dosfstools_repositories()

load("//src/third_party/mtools:mtools_repositories.bzl", "mtools_repositories")
mtools_repositories()
