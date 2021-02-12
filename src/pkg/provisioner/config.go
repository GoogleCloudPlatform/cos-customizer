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

package provisioner

import (
	"encoding/json"
	"fmt"
)

// Config defines a provisioning flow.
type Config struct {
	// BuildContexts identifies the build contexts that should be used during
	// provisioning. A build context means the same thing here as it does
	// elsewhere in cos-customizer. The keys are build context identifiers, and
	// the values are addresses to fetch the build contexts from. Currently, only
	// gs:// addresses are supported.
	BuildContexts map[string]string
	// BootDisk defines how the boot disk should be configured.
	BootDisk struct {
		StatefulSize string
		OEMSize      string
		ReclaimSDA3  string
		VerifiedOEM  bool
	}
	// Steps are provisioning behaviors that can be run.
	// The supported provisioning behaviors are:
	//
	// Type: RunScript
	// Args:
	// - BuildContext: the name of the build contex to run the script in
	// - Path: the path to the script in the build context
	// - Env: Environment variables to pass to the script, in the format
	//   A=B,C=D
	//
	// Type: InstallGPU
	// Args:
	// - Version: The nvidia driver version to install
	// - MD5Sum: An optional md5 hash to use to verify the downloaded nvidia
	//   installer
	// - InstallDir: An absolute path to install nvidia drivers in to
	// - GCSDownloadPrefix: A optional gs:// URI that will be used as a prefix
	//   for downloading cos-gpu-installer dependencies.
	//
	// Type: AppendKernelCmdLine
	// Args:
	// - Value: The exact text to append to the kernel command line.
	Steps []struct {
		Type string
		Args *json.RawMessage
	}
}

type step interface {
	run(*state) error
}

func parseStep(stepType string, stepArgs *json.RawMessage) (step, error) {
	switch stepType {
	case "RunScript":
		var s step
		s = &runScriptStep{}
		if err := json.Unmarshal(*stepArgs, s); err != nil {
			return nil, err
		}
		return s, nil
	default:
		return nil, fmt.Errorf("unknown step type: %q", stepType)
	}
}
