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
	anthosInstallerVersion       = "anthos_installer-linux-amd64-v0.0.1-96f78c3.tar.gz"
	anthosInstallerReleaseBucket = "cos-anthos-builds-us"
)

// InstallPackage installs the packages based on thes
// pkg-spec by the anthos-installer.
type InstallPackage struct {
	PkgSpecURL string
}

// Name implements subcommands.Command.Name.
func (ip *InstallPackage) Name() string {
	return "anthos-installer-install"
}

// Synopsis implements subcommands.Command.Synopsis.
func (ip *InstallPackage) Synopsis() string {
	return "Installs packages specified by pkgspec files."
}

// Usage implements subcommands.Command.Usage.
func (ip *InstallPackage) Usage() string {
	return `anthos-installer-install [flags]
`
}

// SetFlags implements subcommands.Command.SetFlags.
func (ip *InstallPackage) SetFlags(f *flag.FlagSet) {
	f.StringVar(&ip.PkgSpecURL, "pkgspec-url", "", "URL path that points to the package spec.")
}

// Execute implements subcommands.Command.Execute. It configures the current image build process to
// customize the result image with a shell script.
func (ip *InstallPackage) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	if f.NArg() != 0 {
		f.Usage()
		return subcommands.ExitUsageError
	}
	files, ok := args[0].(*fs.Files)
	if !ok {
		log.Panic("InstallPackage expects *fs.Files")
	}

	if ip.PkgSpecURL == "" {
		log.Printf("Required package spec is not provided for %s step\n", ip.Name())
		return subcommands.ExitFailure
	}

	var provConfig provisioner.Config
	if err := config.LoadFromFile(files.ProvConfig, &provConfig); err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}

	buf, err := json.Marshal(&provisioner.InstallPackagesStep{
		BuildContext:                 "user",
		PkgSpecURL:                   ip.PkgSpecURL,
		AnthosInstallerReleaseBucket: anthosInstallerReleaseBucket,
		AnthosInstallerVersion:       anthosInstallerVersion,
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
