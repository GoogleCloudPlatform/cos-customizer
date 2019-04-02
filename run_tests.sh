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
set -o pipefail

PROJECT=""

usage() {
  cat <<'EOF'
Usage: ./run_tests.sh [OPTION]
run_tests.sh runs cos-customizer integration tests.

-p,--project=<project_name>    GCP project to run tests in. Required.
EOF
}

parse_arguments() {
  local -r long_options="project:,help"
  parsed_args="$(getopt --options=p:,h --longoptions="${long_options}" --name "$0" -- "$@")"
  eval set -- "${parsed_args}"
  while true; do
    case "$1" in
      -p|--project)
        PROJECT="$2"
        shift 2
        ;;
      -h|--help)
        usage
        exit
        ;;
      --)
        shift
        break
        ;;
      *)
        usage
        exit
        ;;
    esac
  done
}

get_build_status() {
  local -r build_id="$1"
  gcloud builds describe "${build_id}" --project="${PROJECT}" --format='value(status)'
}

get_log_url() {
  local -r build_id="$1"
  gcloud builds describe "${build_id}" --project="${PROJECT}" --format='value(logUrl)'
}

start_build() {
  local -r config="$1"
  gcloud builds submit --config="${config}" --project="${PROJECT}" --async --format='value(ID)' .
}

wait_for_build() {
  local -r build_id="$1"
  local status
  while true; do
    status=$(get_build_status "${build_id}")
    case "${status}" in
      "SUCCESS"|"FAILURE"|"INTERNAL_ERROR"|"TIMEOUT"|"CANCELLED")
        echo "${status}"
        return
        ;;
      "QUEUED"|"WORKING")
        sleep 5
        ;;
      "STATUS_UNKNOWN")
        echo "Received STATUS_UNKNOWN for build ${build_id}" 1>&2
        sleep 5
        ;;
      *)
        echo "Unknown status for build ${build_id}: ${status}" 1>&2
        return 1
        ;;
    esac
  done
}

main() {
  local -a build_ids
  local status
  local log_url
  local exit_code=0
  if [[ -z "${PROJECT}" ]]; then
    usage
    return 1
  fi
  for config in testing/*.yaml; do
    build_ids+=("$(start_build "${config}")")
  done
  for build_id in "${build_ids[@]}"; do
    status="$(wait_for_build "${build_id}")"
    if [[ "${status}" == "SUCCESS" ]]; then
      echo "Build ${build_id} succeeded"
    else
      log_url="$(get_log_url "${build_id}")"
      echo "Build ${build_id} failed"
      echo "Logs: ${log_url}"
      exit_code=1
    fi
  done
  return "${exit_code}"
}

parse_arguments "$@"
main
