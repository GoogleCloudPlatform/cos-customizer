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

// metadata_watcher is a program that waits for 5 minutes for a specific GCE
// instance metadata key to be present.
package main

import (
	"fmt"
	"os"
	"time"

	"cloud.google.com/go/compute/metadata"
)

func main() {
	if len(os.Args) < 2 {
		os.Exit(1)
	}
	key := os.Args[1]
	fmt.Printf("Waiting for metadata key %q...\n", key)
	end := time.Now().Add(5 * time.Minute)
	for time.Now().Before(end) {
		_, err := metadata.InstanceAttributeValue(key)
		if err == nil {
			fmt.Printf("Found metadata key %q\n", key)
			os.Exit(0)
		}
		fmt.Printf("Could not fetch metadata key %q: %v\n", key, err)
		time.Sleep(time.Second)
	}
}
