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

package cmd

import (
	"context"
	"cos-customizer/fs"
	"flag"
	"fmt"
	"log"

	"github.com/google/subcommands"
)

// DisableAutoUpdate implements subcommands.Command for the "disable-auto-update" command.
// It writes a script name to the state file and run the script in builtin_build_context.
type DisableAutoUpdate struct{}

// Name implements subcommands.Command.Name.
func (d *DisableAutoUpdate) Name() string {
	return "disable-auto-update"
}

// Synopsis implements subcommands.Command.Synopsis.
func (d *DisableAutoUpdate) Synopsis() string {
	return "Disable auto-update service."
}

// Usage implements subcommands.Command.Usage.
func (d *DisableAutoUpdate) Usage() string {
	return `disable-auto-update
`
}

// SetFlags implements subcommands.Command.SetFlags.
func (d *DisableAutoUpdate) SetFlags(f *flag.FlagSet) {}

// Execute implements subcommands.Command.Execute. It disables the auto-update systemd service.
func (d *DisableAutoUpdate) Execute(_ context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	if f.NArg() != 0 {
		f.Usage()
		return subcommands.ExitUsageError
	}
	files := args[0].(*fs.Files)
	if err := fs.AppendStateFile(files.StateFile, fs.Builtin, "disable_auto_update.sh", ""); err != nil {
		log.Println(fmt.Errorf("cannot append state file, error msg:(%v)", err))
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}
