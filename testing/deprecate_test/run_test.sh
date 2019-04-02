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

RESULT="pass"
old_deprecation_state="$(gcloud compute images describe "${OLD_IMAGE}" --project="${PROJECT}" --format='value(deprecated)')"
if [[ -z "${old_deprecation_state}" ]]; then
  echo "Old image isn't deprecated"
  echo "Deprecation state: ${old_deprecation_state}"
  RESULT="fail"
fi

new_deprecation_state="$(gcloud compute images describe "${IMAGE}" --project="${PROJECT}" --format='value(deprecated)')"
if [[ -n "${new_deprecation_state}" ]]; then
  echo "New image appears to be deprecated"
  echo "Deprecation state: ${new_deprecation_state}"
  RESULT="fail"
fi

gcloud compute images delete "${OLD_IMAGE}" --project="${PROJECT}"
gcloud compute images delete "${IMAGE}" --project="${PROJECT}"
if [[ "${RESULT}" == "fail" ]]; then
  echo "Tests failed"
  exit 1
fi
echo "Tests passed"
