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
    sha256 = "2d536797707dd1697441876b2e862c58839f975c8fc2f0f96636cbd428f45866",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.23.5/rules_go-v0.23.5.tar.gz",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.23.5/rules_go-v0.23.5.tar.gz",
    ],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "cdb02a887a7187ea4d5a27452311a75ed8637379a1287d8eeb952138ea485f7d",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.21.1/bazel-gazelle-v0.21.1.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.21.1/bazel-gazelle-v0.21.1.tar.gz",
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

git_repository(
    name = "com_google_protobuf",
    commit = "31ebe2ac71400344a5db91ffc13c4ddfb7589f92",
    remote = "https://github.com/protocolbuffers/protobuf",
)

load("@com_google_protobuf//:protobuf_deps.bzl", "protobuf_deps")

protobuf_deps()

load("@rules_pkg//:deps.bzl", "rules_pkg_dependencies")

load("@io_bazel_rules_go//go:deps.bzl", "go_rules_dependencies", "go_register_toolchains")

go_rules_dependencies()

go_register_toolchains()

rules_pkg_dependencies()

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

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

go_repository(
    name = "com_github_google_subcommands",
    importpath = "github.com/google/subcommands",
    urls = ["https://github.com/google/subcommands/archive/5bae204cdfb2d92dcc333d56014bae6a2f6c58b1.tar.gz"],
    strip_prefix = "subcommands-5bae204cdfb2d92dcc333d56014bae6a2f6c58b1",
    sha256 = "459d1f29a8cb6be068196ead8fc485d54cb895afc257aacaa6d0cab49e3e1fe5",
    type = "tar.gz",
)

go_repository(
    name = "com_google_cloud_go",
    importpath = "cloud.google.com/go",
    # Archives downloaded from gitiles aren't deterministic, so don't compare
    # against a fixed sha256 (https://github.com/google/gitiles/issues/84)
    urls = ["https://code.googlesource.com/gocloud/+archive/13a5d37070fcb4cc601a650c1bcb95885e3cc776.tar.gz"],
    type = "tar.gz",
)

go_repository(
    name = "org_golang_google_grpc",
    importpath = "google.golang.org/grpc",
    urls = ["https://github.com/grpc/grpc-go/archive/v1.30.0.tar.gz"],
    strip_prefix = "grpc-go-1.30.0",
    sha256 = "add9d2c86ea2611a95957ae97c4369c3fc8b381d4c55ca5c0df9daaa220eda54",
    type = "tar.gz",
)

go_repository(
    name = "org_golang_x_net",
    importpath = "golang.org/x/net",
    urls = ["https://github.com/golang/net/archive/ab34263943818b32f575efc978a3d24e80b04bd7.tar.gz"],
    strip_prefix = "net-ab34263943818b32f575efc978a3d24e80b04bd7",
    sha256 = "7f3c48e6aa4dfed3d52639e97d216b9393d31b4c5d6a2da7a898f025d9d41ac7",
    type = "tar.gz",
)

go_repository(
    name = "org_golang_x_text",
    importpath = "golang.org/x/text",
    urls = ["https://github.com/golang/text/archive/23ae387dee1f90d29a23c0e87ee0b46038fbed0e.tar.gz"],
    strip_prefix = "text-23ae387dee1f90d29a23c0e87ee0b46038fbed0e",
    sha256 = "f4bd81a281c3cee2022c4825c4add086d0976563958de9d8bef86d412915ba1e",
    type = "tar.gz",
)

go_repository(
    name = "org_golang_google_api",
    importpath = "google.golang.org/api",
    # Archives downloaded from gitiles aren't deterministic, so don't compare
    # against a fixed sha256 (https://github.com/google/gitiles/issues/84)
    urls = ["https://code.googlesource.com/google-api-go-client/+archive/cb1f45ca288bfafb52ab824361c939d908e525ad.tar.gz"],
    type = "tar.gz",
)

go_repository(
    name = "org_golang_x_oauth2",
    importpath = "golang.org/x/oauth2",
    # Archives downloaded from gitiles aren't deterministic, so don't compare
    # against a fixed sha256 (https://github.com/google/gitiles/issues/84)
    urls = ["https://go.googlesource.com/oauth2/+archive/d2e6202438beef2727060aa7cabdd924d92ebfd9.tar.gz"],
    type = "tar.gz",
)

go_repository(
    name = "in_gopkg_yaml_v2",
    importpath = "gopkg.in/yaml.v2",
    urls = ["https://github.com/go-yaml/yaml/archive/5420a8b6744d3b0345ab293f6fcba19c978f1183.tar.gz"],
    strip_prefix = "yaml-5420a8b6744d3b0345ab293f6fcba19c978f1183",
    sha256 = "f7427a3950b795ae9047c3661e67a7a213f1c1ae9b7efdc1759278473b8d436d",
    type = "tar.gz",
)

go_repository(
    name = "com_github_googleapis_gax_go",
    importpath = "github.com/googleapis/gax-go",
    urls = ["https://github.com/googleapis/gax-go/archive/bd5b16380fd03dc758d11cef74ba2e3bc8b0e8c2.tar.gz"],
    strip_prefix = "gax-go-bd5b16380fd03dc758d11cef74ba2e3bc8b0e8c2",
    sha256 = "6ab51d8764dba1de8bde38a6784340500486052cbf225d41decf9de6230b403a",
    type = "tar.gz",
)

go_repository(
    name = "io_opencensus_go",
    importpath = "go.opencensus.io",
    urls = ["https://github.com/census-instrumentation/opencensus-go/archive/7e6c39beca2921a62fe5f9e53773d750822a6d5c.tar.gz"],
    strip_prefix = "opencensus-go-7e6c39beca2921a62fe5f9e53773d750822a6d5c",
    sha256 = "a31bc593100a4eb8f7364e6fa5f359667acb0f90764430f13877bef84e54d2ee",
    type = "tar.gz",
)

go_repository(
    name = "com_github_google_go-cmp",
    importpath = "github.com/google/go-cmp",
    urls = ["https://github.com/google/go-cmp/archive/875f8df8b7965f1eac1098d36d677f807ac0b49e.tar.gz"],
    strip_prefix = "go-cmp-875f8df8b7965f1eac1098d36d677f807ac0b49e",
    sha256 = "ad74121b3d4d27be6a18818d1daeb5258991c01e4634ab322176f83e858701ec",
    type = "tar.gz",
)

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
    sha256 = "9e7870c3c3b5b0a7f8322c323a3fa641193b1eee792ee7e2eedb6eeebf9969f3",
    snapshot = "20181012T152238Z",
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
    ],
    sources = [
        "@debian_stretch//file:Packages.json",
    ],
)
