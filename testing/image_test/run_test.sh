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

# Image to test
IMAGE="${IMAGE:-}"
PROJECT="${PROJECT:-}"

# Expected values to test against
LICENSES="${LICENSES:-}"
LABELS="${LABELS:-}"
FAMILY="${FAMILY:-}"
DISK_SIZE_GB="${DISK_SIZE_GB:-}"

sort_licenses() {
  local licenses="$1"
  echo "${licenses}" | tr ';' '\n' | sort | tr '\n' ';'
}

RESULT="pass"
actual_licenses="$(gcloud compute images describe "${IMAGE}" --project="${PROJECT}" --format='value(licenses)')"
if [[ "$(sort_licenses "${LICENSES}")" != "$(sort_licenses "${actual_licenses}")" ]]; then
  echo "Licenses differ."
  echo "Expected: ${LICENSES}"
  echo "Actual: ${actual_licenses}"
  RESULT="fail"
fi

actual_labels="$(gcloud compute images describe "${IMAGE}" --project="${PROJECT}" --format='value(labels)')"
if [[ "${LABELS}" != "${actual_labels}" ]]; then
  echo "Labels differ."
  echo "Expected: ${LABELS}"
  echo "Actual: ${actual_labels}"
  RESULT="fail"
fi

actual_family="$(gcloud compute images describe "${IMAGE}" --project="${PROJECT}" --format='value(family)')"
if [[ "${FAMILY}" != "${actual_family}" ]]; then
  echo "Family differs."
  echo "Expected: ${FAMILY}"
  echo "Actual: ${actual_family}"
  RESULT="fail"
fi

actual_disk_size="$(gcloud compute images describe "${IMAGE}" --project="${PROJECT}" --format='value(diskSizeGb)')"
if [[ "${DISK_SIZE_GB}" != "${actual_disk_size}" ]]; then
  echo "Disk size differs."
  echo "Expected: ${DISK_SIZE_GB}"
  echo "Actual: ${actual_disk_size}"
  RESULT="fail"
fi

gcloud compute images delete "${IMAGE}" --project="${PROJECT}"
if [[ "${RESULT}" == "fail" ]]; then
  echo "Tests failed"
  exit 1
fi
echo "Tests passed"
