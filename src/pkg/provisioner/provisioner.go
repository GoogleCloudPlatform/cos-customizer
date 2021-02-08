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

// Package provisioner exports behaviors for provisioning COS systems
// end-to-end. These behaviors are intended to run on a COS system.
package provisioner

import (
	"fmt"
	"os"
)

func cleanup(stateDir string) error {
	return os.RemoveAll(stateDir)
}

// Run runs a full provisioning flow based on the provided config. The stateDir
// is used for persisting data used as part of provisioning. The stateDir allows
// the provisioning flow to be interrupted (e.g. by a reboot) and resumed.
func Run(stateDir string, c Config) (err error) {
	if _, err := initState(stateDir, c); err != nil {
		return err
	}
	// TODO(rkolchmeyer): Implement the actual provisioning behavior
	if err := cleanup(stateDir); err != nil {
		return fmt.Errorf("error in cleanup: %v", err)
	}
	return nil
}
