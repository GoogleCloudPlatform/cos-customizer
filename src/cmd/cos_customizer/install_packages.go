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
	"encoding/json"
	"flag"
	"log"

	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/config"
	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/fs"
	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/provisioner"
	"github.com/google/subcommands"
)

const (
	anthosInstallerVersion = "0.0.1-6f0b777"
)

type InstallPkg struct {
	PkgSpecDir string
	TopWorkDir string
}

// Name implements subcommands.Command.Name.
func (ai *InstallPkg) Name() string {
	return "anthos-installer-install"
}

// Synopsis implements subcommands.Command.Synopsis.
func (ai *InstallPkg) Synopsis() string {
	return "Installs the packages."
}

// Usage implements subcommands.Command.Usage.
func (ai *InstallPkg) Usage() string {
	return `anthos-installer-install [flags]
`
}

// SetFlags implements subcommands.Command.SetFlags.
func (ai *InstallPkg) SetFlags(f *flag.FlagSet) {
	f.StringVar(&ai.PkgSpecDir, "pkg-spec-dir", "", "Path to the directory that has the package spec.")
	f.StringVar(&ai.TopWorkDir, "work-dir", "", "Path to the temporary working directory.")
}

// Execute implements subcommands.Command.Execute. It configures the current image build process to
// customize the result image with a shell script.
func (ai *InstallPkg) Execute(_ context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	if f.NArg() != 0 {
		f.Usage()
		return subcommands.ExitUsageError
	}
	files := args[0].(*fs.Files)
	if ai.PkgSpecDir == "" {
		log.Printf("package spec is not provided for %s step; package spec is required\n", ai.Name())
		return subcommands.ExitFailure
	}
	var provConfig provisioner.Config
	if err := config.LoadFromFile(files.ProvConfig, &provConfig); err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}
	buf, err := json.Marshal(&provisioner.InstallPackagesStep{
		BuildContext:           "user",
		PkgSpecDir:             ai.PkgSpecDir,
		TopWorkDir:             ai.TopWorkDir,
		AnthosInstallerVersion: anthosInstallerVersion,
	})
	if err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}
	provConfig.Steps = append(provConfig.Steps, provisioner.StepConfig{
		Type: "InstallPackages",
		Args: json.RawMessage(buf),
	})
	if err := config.SaveConfigToPath(files.ProvConfig, &provConfig); err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}
