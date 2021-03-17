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
	"context"
	"flag"
	"log"

	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/config"
	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/fs"
	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/provisioner"

	"github.com/google/subcommands"
)

// SealOEM implements subcommands.Command for the "seal-oem" command.
// It builds a hash tree of the OEM partition and modifies the kernel
// command line to verify the OEM partition at boot time.
type SealOEM struct{}

// Name implements subcommands.Command.Name.
func (s *SealOEM) Name() string {
	return "seal-oem"
}

// Synopsis implements subcommands.Command.Synopsis.
func (s *SealOEM) Synopsis() string {
	return "Seal the OEM partition."
}

// Usage implements subcommands.Command.Usage.
func (s *SealOEM) Usage() string {
	return `seal-oem
`
}

// SetFlags implements subcommands.Command.SetFlags.
func (s *SealOEM) SetFlags(f *flag.FlagSet) {}

func (s *SealOEM) updateProvConfig(configPath string) error {
	var provConfig provisioner.Config
	if err := config.LoadFromFile(configPath, &provConfig); err != nil {
		return err
	}
	provConfig.BootDisk.ReclaimSDA3 = true
	provConfig.Steps = append(provConfig.Steps, provisioner.StepConfig{
		Type: "SealOEM",
	})
	return config.SaveConfigToPath(configPath, &provConfig)
}

// Execute implements subcommands.Command.Execute. It modifies the kernel command line
// to enable dm-verity check on /dev/sda8 and disables update-engine (auto-update) and
// usr-share-oem-mount systemd services.
func (s *SealOEM) Execute(_ context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	if f.NArg() != 0 {
		f.Usage()
		return subcommands.ExitUsageError
	}
	files := args[0].(*fs.Files)
	if err := s.updateProvConfig(files.ProvConfig); err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}
