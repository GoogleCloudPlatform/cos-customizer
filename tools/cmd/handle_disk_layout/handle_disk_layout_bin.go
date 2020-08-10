// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the License);
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an AS IS BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"cos-customizer/tools"
	"log"
	"os"
	"strconv"
)

// main generates binary file to extend the OEM partition.
// Built by Bazel. The binary will be in data/builtin_build_context/.
func main() {
	log.SetOutput(os.Stdout)
	args := os.Args
	if len(args) != 6 {
		log.Fatalln("error: must have 5 arguments: disk string, statePartNum, oemPartNum int, oemSize string, reclaimSDA3 bool")
	}
	statePartNum, err := strconv.Atoi(args[2])
	if err != nil {
		log.Fatalln("error: the 2nd argument statePartNum must be an int")
	}
	oemPartNum, err := strconv.Atoi(args[3])
	if err != nil {
		log.Fatalln("error: the 3rd argument oemPartNum must be an int")
	}
	reclaimSDA3, err := strconv.ParseBool(args[5])
	if err != nil {
		log.Fatalln("error: the 5th argument reclaimSDA3 must be a bool")
	}
	if err := tools.HandleDiskLayout(args[1], statePartNum, oemPartNum, args[4], reclaimSDA3); err != nil {
		log.Fatalln(err)
	}
}
