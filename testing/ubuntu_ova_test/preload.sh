#!/bin/bash
#
# Copyright 2021 Google LLC
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
set -o pipefail

install_docker() {
    apt-get update
    apt-get install -y \
      apt-transport-https \
      ca-certificates \
      curl \
      gnupg \
      lsb-release
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
      | gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
    echo \
      "deb [arch=amd64 signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu \
      $(lsb_release -cs) stable" | tee /etc/apt/sources.list.d/docker.list > /dev/null
    apt-get update
    apt-get install -y docker-ce docker-ce-cli containerd.io
}

install_docker_with_retry() {
  local err_code=0
  local i=1
  while [[ $i -le 10 ]]; do
    echo "Installing docker... [${i}/10]"
    install_docker && break || err_code="$?"
    ((i++))
  done
  if [[ $i -eq 11 ]]; then
    echo "Failed to install docker."
    return "${err_code}"
  fi
  echo "Successfully installed docker."
}

docker_pull_ubuntu() {
  local docker_code=0
  local i=1
  while [[ $i -le 10 ]]; do
    echo "Pulling ubuntu container image... [${i}/10]"
    docker pull ubuntu && break || docker_code="$?"
    ((i++))
  done
  if [[ $i -eq 11 ]]; then
    echo "Pulling ubuntu failed."
    echo "Docker journal logs:"
    journalctl -u docker.service --no-pager
    return "${docker_code}"
  fi
  echo "Successfully pulled ubuntu container image."
}

echo "hello" > /var/lib/hello
install_docker_with_retry
docker_pull_ubuntu
