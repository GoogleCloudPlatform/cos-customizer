#!/bin/bash
#
# Copyright 2018 Google LLC
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

# Templated by cos-customizer. Since the functionality provided by
# this script needs to be run every time the system boots, and so needs
# to be installed persistently on the system, it's simpler to keep the script
# completely self-contained.
export NVIDIA_DRIVER_VERSION={{.NvidiaDriverVersion}}
export NVIDIA_DRIVER_MD5SUM={{.NvidiaDriverMd5sum}}
export NVIDIA_INSTALL_DIR_HOST={{.NvidiaInstallDirHost}}
export COS_NVIDIA_INSTALLER_CONTAINER=gcr.io/cos-cloud/cos-gpu-installer:v20181106
export NVIDIA_INSTALL_DIR_CONTAINER=/usr/local/nvidia
export ROOT_MOUNT_DIR=/root


main() {
  mkdir -p "${NVIDIA_INSTALL_DIR_HOST}"
  mount --bind "${NVIDIA_INSTALL_DIR_HOST}" "${NVIDIA_INSTALL_DIR_HOST}"
  mount -o remount,exec "${NVIDIA_INSTALL_DIR_HOST}"
  docker run \
    --rm \
    --privileged \
    --net=host \
    --pid=host \
    --volume "${NVIDIA_INSTALL_DIR_HOST}":"${NVIDIA_INSTALL_DIR_CONTAINER}" \
    --volume /dev:/dev \
    --volume "/":"${ROOT_MOUNT_DIR}" \
    -e NVIDIA_DRIVER_VERSION \
    -e NVIDIA_DRIVER_MD5SUM \
    -e NVIDIA_INSTALL_DIR_HOST \
    -e COS_NVIDIA_INSTALLER_CONTAINER \
    -e NVIDIA_INSTALL_DIR_CONTAINER \
    -e ROOT_MOUNT_DIR \
    "${COS_NVIDIA_INSTALLER_CONTAINER}"
  ${NVIDIA_INSTALL_DIR_HOST}/bin/nvidia-smi

  # Start nvidia-persistenced
  if ! pgrep -f nvidia-persistenced > /dev/null; then
    "${NVIDIA_INSTALL_DIR_HOST}/bin/nvidia-persistenced" --verbose
  fi

  # Set softlockup_panic
  echo 1 > /proc/sys/kernel/softlockup_panic

  # Install this script as a setup script for future system boots
  if [[ "$(realpath "${BASH_SOURCE[0]}")" != "${NVIDIA_INSTALL_DIR_HOST}/setup_gpu.sh" ]]; then
    cp "$(realpath "${BASH_SOURCE[0]}")" "${NVIDIA_INSTALL_DIR_HOST}/setup_gpu.sh"
  fi
}

main
