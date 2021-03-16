# Copyright 2021 Google LLC
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
load("@bazel_tools//tools/build_defs/repo:utils.bzl", "maybe")

def mtools_repositories():
    """Load all repositories needed for mtools."""

    maybe(
        http_archive,
        name = "mtools",
        build_file = Label("//src/third_party/mtools:BUILD.mtools.bazel"),
        strip_prefix = "mtools-4.0.26",
        urls = [
            "https://mirror.bazel.build/ftp.gnu.org/gnu/mtools/mtools-4.0.26.tar.gz",
            "http://ftp.gnu.org/gnu/mtools/mtools-4.0.26.tar.gz",
        ],
        sha256 = "b1adb6973d52b3b70b16047e682f96ef1b669d6b16894c9056a55f407e71cd0f",
    )
