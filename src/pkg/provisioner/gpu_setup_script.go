// Copyright 2021 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package provisioner

const gpuSetupScriptTemplate = `#!/bin/bash

set -o errexit

export NVIDIA_DRIVER_VERSION={{.NvidiaDriverVersion}}
export NVIDIA_DRIVER_MD5SUM={{.NvidiaDriverMD5Sum}}
export NVIDIA_INSTALL_DIR_HOST={{.NvidiaInstallDirHost}}
export COS_NVIDIA_INSTALLER_CONTAINER={{.NvidiaInstallerContainer}}
export NVIDIA_INSTALL_DIR_CONTAINER=/usr/local/nvidia
export ROOT_MOUNT_DIR=/root

pull_installer() {
  local docker_code
  local i=1
  while [[ $i -le 10 ]]; do
    echo "Pulling cos-gpu-installer container image... [${i}/10]"
    docker pull "${COS_NVIDIA_INSTALLER_CONTAINER}" && break || docker_code="$?"
    i=$((i+1))
    sleep 2
  done
  if [[ $i -eq 11 ]]; then
    echo "Pulling cos-gpu-installer failed."
    echo "Docker journal logs:"
    journalctl -u docker.service --no-pager
    exit "${docker_code}"
  fi
  echo "Successfully pulled cos-gpu-installer container image."
}

main() {
  mkdir -p "${NVIDIA_INSTALL_DIR_HOST}"
  mount --bind "${NVIDIA_INSTALL_DIR_HOST}" "${NVIDIA_INSTALL_DIR_HOST}"
  mount -o remount,exec "${NVIDIA_INSTALL_DIR_HOST}"
  pull_installer
  docker_run_cmd="docker run \
    --rm \
    --privileged \
    --net=host \
    --pid=host \
    --volume ${NVIDIA_INSTALL_DIR_HOST}:${NVIDIA_INSTALL_DIR_CONTAINER} \
    --volume /dev:/dev \
    --volume /:${ROOT_MOUNT_DIR} \
    -e NVIDIA_DRIVER_VERSION \
    -e NVIDIA_DRIVER_MD5SUM \
    -e NVIDIA_INSTALL_DIR_HOST \
    -e COS_NVIDIA_INSTALLER_CONTAINER \
    -e NVIDIA_INSTALL_DIR_CONTAINER \
    -e ROOT_MOUNT_DIR \
    -e COS_DOWNLOAD_GCS \
    -e GPU_INSTALLER_DOWNLOAD_URL \
    ${COS_NVIDIA_INSTALLER_CONTAINER}"
  if ! ${docker_run_cmd}; then
    echo "GPU install failed."
    if [[ -f /var/lib/nvidia/nvidia-installer.log ]]; then
      echo "Nvidia installer debug logs:"
      cat /var/lib/nvidia/nvidia-installer.log
    fi
    return 1
  fi
  ${NVIDIA_INSTALL_DIR_HOST}/bin/nvidia-smi

  # Start nvidia-persistenced
  if ! pgrep -f nvidia-persistenced > /dev/null; then
    "${NVIDIA_INSTALL_DIR_HOST}/bin/nvidia-persistenced" --verbose
  fi

  # Set softlockup_panic
  echo 1 > /proc/sys/kernel/softlockup_panic
}

main
`
