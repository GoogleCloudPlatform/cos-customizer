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

type StepConfig struct {
	Type string
	Args json.RawMessage
}

type BootDiskConfig struct {
	OEMSize           string
	OEMFSSize4K       uint64
	ReclaimSDA3       bool
	WaitForDiskResize bool
}

// Config defines a provisioning flow.
type Config struct {
	// BuildContexts identifies the build contexts that should be used during
	// provisioning. A build context means the same thing here as it does
	// elsewhere in cos-customizer. The keys are build context identifiers, and
	// the values are addresses to fetch the build contexts from. Currently, only
	// gs:// addresses are supported.
	BuildContexts map[string]string
	// BootDisk defines how the boot disk should be configured.
	BootDisk BootDiskConfig
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
	// - NvidiaDriverVersion: The nvidia driver version to install. Can also be
	//   the name of an nvidia installer .run file. If a .run file is provided and
	//   a GCSDepsPrefix is provided, the .run file will be fetched from the
	//   GCSDepsPrefix location.
	// - NvidiaDriverMD5Sum: An optional md5 hash to use to verify the downloaded nvidia
	//   installer.
	// - NvidiaInstallDirHost: An absolute path specifying where nvidia drivers
	//   should be installed. Defaults to /var/lib/nvidia.
	// - NvidiaInstallerContainer: The cos-gpu-installer container image to use
	//   for installing nvidia drivers.
	// - GCSDepsPrefix: A optional gs:// URI that will be used as a prefix
	//   for downloading cos-gpu-installer dependencies.
	//
	// Type: DisableAutoUpdate
	// Args: This step takes no arguments.
	//
	// Type: SealOEM
	// Args: This step takes no arguments.
	Steps []StepConfig
}

type step interface {
	run(*state) error
}

func parseStep(stepType string, stepArgs json.RawMessage) (step, error) {
	switch stepType {
	case "RunScript":
		var s step
		s = &RunScriptStep{}
		if err := json.Unmarshal(stepArgs, s); err != nil {
			return nil, err
		}
		return s, nil
	case "InstallGPU":
		var s step
		s = &InstallGPUStep{}
		if err := json.Unmarshal(stepArgs, s); err != nil {
			return nil, err
		}
		return s, nil
	case "DisableAutoUpdate":
		return &DisableAutoUpdateStep{}, nil
	case "SealOEM":
		return &SealOEMStep{}, nil
	default:
		return nil, fmt.Errorf("unknown step type: %q", stepType)
	}
}
