#!/bin/bash
#
# Copyright 2020 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -o errexit

# get user input from metadata
oem_fs_size_4k="$(/usr/share/google/get_metadata_value \
                attributes/OEMFSSize4K)"
sudo mount -o remount,exec /var
sudo chmod 777 ./seal_oem.bin
sudo ./seal_oem.bin "${oem_fs_size_4k}"
