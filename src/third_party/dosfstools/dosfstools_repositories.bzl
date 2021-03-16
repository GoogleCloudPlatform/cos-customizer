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

def dosfstools_repositories():
    """Load all repositories needed for dosfstools."""

    maybe(
        http_archive,
        name = "dosfstools",
        build_file = Label("//src/third_party/dosfstools:BUILD.dosfstools.bazel"),
        strip_prefix = "dosfstools-4.2",
        urls = ["https://github.com/dosfstools/dosfstools/releases/download/v4.2/dosfstools-4.2.tar.gz"],
        sha256 = "64926eebf90092dca21b14259a5301b7b98e7b1943e8a201c7d726084809b527",
    )
