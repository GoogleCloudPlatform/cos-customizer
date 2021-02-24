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

package main

import (
	"context"
	"errors"
	"flag"
	"log"

	"github.com/google/subcommands"

	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/provisioner"
)

// Resume implements subcommands.Command for the "resume" command.
// This command resumes provisioning from given provisioning state.
type Resume struct{}

// Name implements subcommands.Command.Name.
func (r *Resume) Name() string {
	return "resume"
}

// Synopsis implements subcommands.Command.Synopsis.
func (r *Resume) Synopsis() string {
	return "Resume provisioning from provided state. Has an exit code of 3 if a reboot is required after execution."
}

// Usage implements subcommands.Command.Usage.
func (r *Resume) Usage() string {
	return `resume
`
}

// SetFlags implements subcommands.Command.SetFlags.
func (r *Resume) SetFlags(f *flag.FlagSet) {}

// Execute implements subcommands.Command.Execute.
func (r *Resume) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	deps := args[0].(provisioner.Deps)
	exitCode := args[1].(*int)
	if err := provisioner.Resume(ctx, deps, *stateDir); err != nil {
		if errors.Is(err, provisioner.ErrRebootRequired) {
			log.Println(rebootMsg)
			*exitCode = 3
			return subcommands.ExitSuccess
		}
		log.Printf("Provisioning error: %v", err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}
