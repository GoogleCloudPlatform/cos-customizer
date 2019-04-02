# Copyright 2018 Google LLC
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

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "7be7dc01f1e0afdba6c8eb2b43d2fa01c743be1b9273ab1eaf6c233df078d705",
    url = "https://github.com/bazelbuild/rules_go/releases/download/0.16.5/rules_go-0.16.5.tar.gz",
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "3c681998538231a2d24d0c07ed5a7658cb72bfb5fd4bf9911157c0e9ac6a2687",
    urls = ["https://github.com/bazelbuild/bazel-gazelle/releases/download/0.17.0/bazel-gazelle-0.17.0.tar.gz"],
)

http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "29d109605e0d6f9c892584f07275b8c9260803bf0c6fcb7de2623b2bedc910bd",
    strip_prefix = "rules_docker-0.5.1",
    urls = ["https://github.com/bazelbuild/rules_docker/archive/v0.5.1.tar.gz"],
)

http_archive(
    name = "distroless",
    sha256 = "f7a6ecfb8174a1dd4713ea3b21621072996ada7e8f1a69e6ae7581be137c6dd6",
    strip_prefix = "distroless-446923c3756ceeaa75888f52fcbdd48bb314fbf8",
    urls = ["https://github.com/GoogleContainerTools/distroless/archive/446923c3756ceeaa75888f52fcbdd48bb314fbf8.tar.gz"],
)

load("@io_bazel_rules_go//go:def.bzl", "go_rules_dependencies", "go_register_toolchains")

go_rules_dependencies()

go_register_toolchains()

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")

gazelle_dependencies()

load(
    "@io_bazel_rules_docker//container:container.bzl",
    "container_pull",
    container_repositories = "repositories",
)

container_repositories()

container_pull(
    name = "daisy",
    digest = "sha256:ffbd78eb5390a6fd7be43ec393d8f1e1a6448ea1bad23787f47028f3ada48926",
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
    commit = "5bae204cdfb2d92dcc333d56014bae6a2f6c58b1",
    importpath = "github.com/google/subcommands",
)

go_repository(
    name = "com_google_cloud_go",
    commit = "13a5d37070fcb4cc601a650c1bcb95885e3cc776",
    importpath = "cloud.google.com/go",
)

go_repository(
    name = "org_golang_google_api",
    commit = "6142e720c068c6cd71f2258e007ff1991572e1d5",
    importpath = "google.golang.org/api",
)

go_repository(
    name = "org_golang_x_oauth2",
    commit = "d2e6202438beef2727060aa7cabdd924d92ebfd9",
    importpath = "golang.org/x/oauth2",
)

go_repository(
    name = "in_gopkg_yaml_v2",
    commit = "5420a8b6744d3b0345ab293f6fcba19c978f1183",
    importpath = "gopkg.in/yaml.v2",
)

go_repository(
    name = "com_github_googleapis_gax_go",
    commit = "1ef592c90f479e3ab30c6c2312e20e13881b7ea6",
    importpath = "github.com/googleapis/gax-go",
)

go_repository(
    name = "io_opencensus_go",
    commit = "7e6c39beca2921a62fe5f9e53773d750822a6d5c",
    importpath = "go.opencensus.io",
)

go_repository(
    name = "com_github_google_go-cmp",
    commit = "875f8df8b7965f1eac1098d36d677f807ac0b49e",
    importpath = "github.com/google/go-cmp",
)

load(
    "@distroless//package_manager:package_manager.bzl",
    "package_manager_repositories",
    "dpkg_src",
    "dpkg_list",
)

package_manager_repositories()

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
