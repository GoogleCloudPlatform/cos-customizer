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
#
# This script runs customization scripts on a COS VM instance. It pulls
# source from GCS and executes it.

set -o errexit
set -o pipefail
set -o nounset


PYTHON_IMG="python:2.7.15-alpine"
SDA3_CHECK_FILE="/mnt/stateful_partition/sda3_check"
LAYOUT_CHECK_FILE="/mnt/stateful_partition/layout_check"
INPUT_DEV_NAME="persistent-disk-0"

fatal() {
  echo -e "BuildFailed: ${*}"
  exit 1
}

switch_root(){
  # Get ReclaimSDA3 input from metadata.
  # If needed, switch root partition to /dev/sda5 (rootB)
  local -r reclaim="$(/usr/share/google/get_metadata_value \
    attributes/ReclaimSDA3)"
  echo "Checking the need to switch root partition..."
  if [[ "${reclaim}" == false ]] || [[ "$( sudo rootdev -s | grep sda5)" ]]; then
    echo "Done switching root partition."
    return
  fi
  echo "Copying sda3 to sda5..."
  dd if="/dev/sda3" of="/dev/sda5" bs=4M
  echo "sda3 copied to sda5."
  sudo cgpt prioritize -P 5 -i 4 /dev/sda
  # overwrite trap to avoid build failure triggered by reboot.
  trap - EXIT
  echo "Rebooting..."
  reboot
  # keep it inside of this function until reboot kills the process
  while :
    do
      sleep 1
    done
}

enter_workdir() {
  echo "Entering working directory..."
  mkdir -p /var/lib/.cos-customizer
  cd /var/lib/.cos-customizer
  echo "Finished entering working directory"
}

exit_workdir() {
  echo "Exiting and cleaning up working directory..."
  cd /root
  rm -rf /var/lib/.cos-customizer
  echo "Finished exiting working directory"
}

setup() {
  echo "Setting up the environment for preloading..."
  stop_service update-engine
  mount -t tmpfs tmpfs /root
  docker-credential-gcr configure-docker
  echo "Done setting up the environment for preloading"
}

pull_python() {
  echo "Getting ready to pull python container image"
  local docker_code
  local i=1
  while [[ $i -le 10 ]]; do
    echo "Pulling python container image... [${i}/10]"
    docker pull "${PYTHON_IMG}" && break || docker_code="$?"
    i=$((i+1))
  done
  if [[ $i -eq 11 ]]; then
    echo "Pulling python failed."
    echo "Docker journal logs:"
    journalctl -u docker.service --no-pager
    exit "${docker_code}"
  fi
  echo "Successfully pulled python container image."
}

download_gcs_object() {
  pull_python
  local -r url="$1"
  echo "Downloading GCS object ${url}..."
  local -r bucket="$(echo "${url#gs://}" | cut -d/ -f 1)"
  local -r object="$(echo "${url#gs://}" | cut -d/ -f 2-)"
  local -r encoded_object="$(docker run --rm "${PYTHON_IMG}" \
    python -c "import urllib; print(urllib.quote('''${object}''', safe=''))")"
  local -r creds="$(/usr/share/google/get_metadata_value \
    service-accounts/default/token)"
  local -r access_token="$(echo "${creds}" | docker run --rm -i "${PYTHON_IMG}" \
    python -c "import sys; import json; print(json.loads(sys.stdin.read())['access_token'])")"
  curl -X GET \
    --retry 5 \
    -H "Authorization: Bearer ${access_token}" \
    -o "$(basename "${object}")" \
    "https://www.googleapis.com/storage/v1/b/${bucket}/o/${encoded_object}?alt=media"
  echo "Done downloading ${url}"
  basename "${object}"
}

wait_daisy_logging() {
  if [[ -e "daisy_ack" && "$(cat daisy_ack)" == "ack" ]]; then
    echo "getSerialPortOutput is healthy"
    return
  fi
  local -r gcs_path=$(/usr/share/google/get_metadata_value \
    attributes/DaisyAck)
  until [[ "$(cat "$(download_gcs_object "${gcs_path}" | tail -n 1)")" == "ack" ]]; do
    echo "Waiting for ack from Daisy that getSerialPortOutput is healthy..."
    sleep 2
  done
  echo "getSerialPortOutput is healthy"
}

fetch_user_ctx() {
  if [[ -e "user_ctx_dir" ]]; then
    echo "user build context already exists"
    return
  fi
  echo "Fetching user build context..."
  local -r user_ctx_gcs="$(/usr/share/google/get_metadata_value \
    attributes/UserBuildContext)"
  local -r user_ctx="$(download_gcs_object "${user_ctx_gcs}" | tail -n 1)"
  mkdir user_ctx_dir
  if [[ -s "${user_ctx}" ]]; then
    tar xvf "${user_ctx}" -C user_ctx_dir
  fi
  echo "Done fetching user build context"
}

fetch_builtin_ctx() {
  if [[ -e "builtin_ctx_dir" ]]; then
    echo "builtin build context already exists"
    return
  fi
  echo "Fetching builtin build context..."
  local -r builtin_ctx_gcs="$(/usr/share/google/get_metadata_value \
    attributes/BuiltinBuildContext)"
  local -r builtin_ctx="$(download_gcs_object "${builtin_ctx_gcs}" | tail -n 1)"
  mkdir builtin_ctx_dir
  if [[ -s "${builtin_ctx}" ]]; then
    tar xvf "${builtin_ctx}" -C builtin_ctx_dir
  fi
  echo "Done fetching builtin build context"
}

# need to forcefully stop journald service to unmount stateful partition.
stop_journald_service() {
  mkdir -p /etc/systemd/system/systemd-journald.service.d
  cat > /etc/systemd/system/systemd-journald.service.d/override.conf<<EOF
[Service]
Restart=no
EOF

  systemctl daemon-reload
  stop_service systemd-journald.socket 
  stop_service systemd-journald-dev-log.socket 
  stop_service systemd-journald-audit.socket 
  stop_service syslog.socket 
  stop_service systemd-journald.service 
}

# this unit runs at shutdown time after everything but /tmp is unmounted
create_run_after_unmount_unit(){
  mount -o remount,exec /tmp
  # get user input from metadata
  local -r oem_size="$(/usr/share/google/get_metadata_value \
    attributes/OEMSize)"
  local -r reclaim="$(/usr/share/google/get_metadata_value \
    attributes/ReclaimSDA3)"
  cat > /etc/systemd/system/last-run.service<<EOF
[Unit]
Description=Run after everything unmounted
DefaultDependencies=false
Conflicts=shutdown.target
Before=mnt-stateful_partition.mount usr-share-oem.mount
After=tmp.mount

[Service]
Type=oneshot
RemainAfterExit=true
ExecStart=/bin/true
ExecStop=/bin/bash -c '/tmp/handle_disk_layout.bin /dev/sda 1 8 "${oem_size}" "${reclaim}" |sed "s/^/BuildStatus: /"'
TimeoutStopSec=600
StandardOutput=tty
StandardError=tty
TTYPath=/dev/ttyS2
EOF
}

handle_disk_layout(){
  echo "Modifying disk layout..."
  create_run_after_unmount_unit
  cp builtin_ctx_dir/handle_disk_layout.bin /tmp/handle_disk_layout.bin
  systemctl --no-block start last-run.service
  stop_journald_service
  echo "Rebooting..."
  
  # overwrite trap to avoid build failure triggered by reboot.
  trap - EXIT
  reboot
  # keep it inside of this function until reboot kills the process
  while :
    do
      sleep 1
    done
}

fetch_state_file() {
  if [[ -e "state_file" ]]; then
    echo "state file already exists"
    return
  fi
  echo "Fetching state file..."
  local -r state_file_gcs="$(/usr/share/google/get_metadata_value \
    attributes/StateFile)"
  local -r state_file="$(download_gcs_object "${state_file_gcs}" | tail -n 1)"
  if [[ "${state_file}" != "state_file" ]]; then
    mv "${state_file}" state_file
  fi
  echo "Done fetching state file"
}

# Executes a state file instruction. State file instructions have the following
# format:
# <build_context>\t<script>\t<env>\n
execute_instr() {
  local line="$1"
  local ctx
  local script
  local env
  echo "Executing instruction ${line}..."
  ctx="$(echo -e "${line}" | cut -f 1)"
  script="$(echo -e "${line}" | cut -f 2)"
  env="$(echo -e "${line}" | cut -f 3)"
  case "${ctx}" in
  "user")
    pushd user_ctx_dir
    echo "Executing user script ${script}"
    if [[ ! -z "${env}" ]]; then
      env="../builtin_ctx_dir/${env}"
    fi
    ;;
  "builtin")
    pushd builtin_ctx_dir
    echo "Executing builtin script ${script}"
    ;;
  *)
    echo "Cannot find build context: ${ctx}"
    exit 1
  esac
  if [[ ! -z "${env}" ]]; then
    echo "Using the following environment:"
    cat "${env}"
    (set -o errexit; . "$(realpath "${env}")"; /bin/bash "$(realpath "${script}")")
  else
    (/bin/bash "$(realpath "${script}")")
  fi
  echo "Finished running script ${script}."
  popd
  echo "Done executing instruction ${line}"
}

execute_state_file() {
  echo "Running preload scripts..."
  while true; do
    local line
    line="$(head -n 1 state_file)"
    if [[ -z "$line" ]]; then
      break
    fi
    execute_instr "$line"
    sed -i -e "1d" state_file
  done
  echo "Done running preload scripts."
}

stop_service() {
  local -r name="$1"
  # We don't want to call `systemctl stop` on a unit that doesn't exist.
  # `systemctl is-active` is a good enough proxy for that, so let's use that to
  # avoid calling `systemctl stop` on a unit that doesn't exist.
  if systemctl -q is-active "${name}"; then
    echo "${name} is active, stopping..."
    systemctl stop "${name}"
    echo "${name} stopped"
  else
    echo "${name} is not active, ignoring"
  fi
}

stop_services() {
  echo "Stopping services..."
  stop_service crash-reporter
  stop_service crash-sender
  stop_service device_policy_manager
  stop_service metrics-daemon
  stop_service update-engine
  echo "Done stopping services."
}

cleanup() {
  echo "Cleaning up instance state..."
  rm -rf /mnt/stateful_partition/etc
  rm -rf /var/cache/*
  find /var/log -type f -exec cp /dev/null {} \;
  rm -rf /var/tmp/*
  rm -rf /var/lib/crash_reporter/*
  rm -rf /var/lib/metrics/*
  rm -rf /var/lib/systemd/*
  rm -rf /var/lib/update_engine/*
  rm -rf /var/lib/whitelist/*
  rm -f "${SDA3_CHECK_FILE}"
  rm -f "${LAYOUT_CHECK_FILE}"
  echo "Done cleaning up instance state."
}

shrink_sda3(){
  # get user input from metadata
  local -r reclaim="$(/usr/share/google/get_metadata_value \
    attributes/ReclaimSDA3)"

  echo "Checking whether need to shrink sda3..."
  # have shrinked sda3 or no need to shrink sda3
  if [[ -e "${SDA3_CHECK_FILE}" || "${reclaim}" == false ]]; then
    echo "Completed dealing with sda3."
    return
  fi
  echo "Shrinking sda3..."
  touch "${SDA3_CHECK_FILE}"
  handle_disk_layout
}

move_partitions(){
  # get user input from metadata
  local -r oem_size="$(/usr/share/google/get_metadata_value \
    attributes/OEMSize)"
  local -r oem_fs_size_4k="$(/usr/share/google/get_metadata_value \
    attributes/OEMFSSize4K)"
  local -r reclaim="$(/usr/share/google/get_metadata_value \
    attributes/ReclaimSDA3)"

  echo "Checking whether need to change disk layout..."
  if [[ -z "${oem_size}" ]] && [[ "${reclaim}" == false ]]; then 
    echo "No request to change disk layout."
    return
  fi
  # have changed disk layout.
  if [[ -e "${LAYOUT_CHECK_FILE}" ]]; then
    # oem extended, need to resize the file system.
    if [[ -n "${oem_size}" ]]; then
      echo "Resizing OEM partition file system..."
      umount /dev/sda8
      e2fsck -fp /dev/sda8
      if [[ "${oem_fs_size_4k}" -eq "0" ]]; then
        resize2fs /dev/sda8
      else
        resize2fs /dev/sda8 "${oem_fs_size_4k}"
      fi
      systemctl start usr-share-oem.mount
    fi
    fdisk -l
    df -h
    echo "Successfully modified disk layout."
  else
    touch "${LAYOUT_CHECK_FILE}"
    handle_disk_layout
  fi
}

# prepare runs before Daisy step `resize-disk`
prepare() {
  trap 'fatal exiting due to errors' EXIT
  switch_root
  enter_workdir
  setup
  fetch_builtin_ctx
  shrink_sda3
  echo "Preparation done."
}

# build runs after Daisy step `resize-disk`
build() {
  trap 'fatal exiting due to errors' EXIT
  enter_workdir
  wait_daisy_logging
  echo "Downloading source artifacts from GCS..."
  move_partitions
  fetch_user_ctx
  fetch_state_file
  docker rmi "${PYTHON_IMG}" || :
  echo "Successfully downloaded source artifacts from GCS."
  echo "Preparing to run preload scripts..."
  execute_state_file
  exit_workdir
  stop_services
  cleanup
}

prepare 2>&1 | sed "s/^/PrepareStatus: /"
build 2>&1 | sed "s/^/BuildStatus: /"
trap - EXIT
echo "BuildSucceeded: Build completed with no errors. Shutting down..."
# We tell Daisy to check for serial logs every 2 seconds (see
# data/build_image.wf.json). However, sometimes Daisy checks for logs
# every 4-6 seconds. Sleep gives Daisy time to grab the serial logs
# even when it is slow.
sleep 15 || fatal "sleep returned non-zero error code $?"
shutdown -h now || fatal "shutdown returned non-zero error code $?"
