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
from gcr.io/compute-image-tools/daisy as daisy
from ubuntu
run apt-get update && apt-get install -y ca-certificates
copy --from=daisy /daisy /daisy
copy --from=daisy /workflows /workflows
copy ./run_test.wf.json /
copy ./vm.wf.json /
copy  ./preload_test.cfg /
