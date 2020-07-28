// Copyright 2020 Google LLC
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

package main

import (
	"cos-customizer/tools"
	"log"
	"os"
	"strconv"
)

// main generates binary file to seal the OEM partition.
// Built by Bazel. The binary will be in data/builtin_build_context/.
func main() {
	log.SetOutput(os.Stdout)
	args := os.Args
	if len(args) != 2 {
		log.Fatalln("error: must have 1 argument: oemFSSize4K uint64")
	}
	oemFSSize4K, err := strconv.ParseUint(args[1], 10, 64)
	if err != nil {
		log.Fatalln("error: the argument oemFSSize4K must be an uint64")
	}
	if err := tools.SealOEMPartition(oemFSSize4K); err != nil {
		log.Fatalln(err.Error())
	}
}
