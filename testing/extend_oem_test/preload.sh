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

echo "hello" > /mnt/stateful_partition/hello
echo "hello" > /usr/share/oem/hello
docker_code=0
i=1
while [[ $i -le 10 ]]; do
  echo "Pulling ubuntu container image... [${i}/10]"
  docker pull ubuntu && break || docker_code="$?"
  i=$((i+1))
done
if [[ $i -eq 11 ]]; then
  echo "Pulling ubuntu failed."
  echo "Docker journal logs:"
  journalctl -u docker.service --no-pager
  exit "${docker_code}"
fi
echo "Successfully pulled ubuntu container image."
