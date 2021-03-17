// Copyright 2018 Google LLC
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
	"sort"
	"strings"

	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/config"
	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/fs"
	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/provisioner"

	"github.com/google/subcommands"
)

// RunScript implements subcommands.Command for the "run-script" command.
// This command configures the current image build process to customize the result image
// with a shell script.
type RunScript struct {
	script string
	env    *mapVar
}

// Name implements subcommands.Command.Name.
func (r *RunScript) Name() string {
	return "run-script"
}

// Synopsis implements subcommands.Command.Synopsis.
func (r *RunScript) Synopsis() string {
	return "Configure the image build with a script to run."
}

// Usage implements subcommands.Command.Usage.
func (r *RunScript) Usage() string {
	return `run-script [flags]
`
}

// SetFlags implements subcommands.Command.SetFlags.
func (r *RunScript) SetFlags(f *flag.FlagSet) {
	f.StringVar(&r.script, "script", "", "Name of script to run.")
	if r.env == nil {
		r.env = newMapVar()
	}
	f.Var(r.env, "env", "Env vars to set before running the script.")
}

// createEnvString creates an environment variable string used by the
// provisioner tool. The format is the same as the format used by exec.Command.
// Elements are sorted for predictable output.
func createEnvString(m map[string]string) string {
	var elems []string
	for k, v := range m {
		elems = append(elems, k+"="+v)
	}
	sort.Strings(elems)
	return strings.Join(elems, ",")
}

// Execute implements subcommands.Command.Execute. It configures the current image build process to
// customize the result image with a shell script.
func (r *RunScript) Execute(_ context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	if f.NArg() != 0 {
		f.Usage()
		return subcommands.ExitUsageError
	}
	files := args[0].(*fs.Files)
	if r.script == "" {
		log.Printf("script not provided for %s step; script is required\n", r.Name())
		return subcommands.ExitFailure
	}
	isValid, err := fs.ArchiveHasObject(files.UserBuildContextArchive, r.script)
	if err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}
	if !isValid {
		log.Printf("could not find script %s in build context", r.script)
		return subcommands.ExitFailure
	}
	var provConfig provisioner.Config
	if err := config.LoadFromFile(files.ProvConfig, &provConfig); err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}
	buf, err := json.Marshal(&provisioner.RunScriptStep{
		BuildContext: "user",
		Path:         r.script,
		Env:          createEnvString(r.env.m),
	})
	if err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}
	provConfig.Steps = append(provConfig.Steps, provisioner.StepConfig{
		Type: "RunScript",
		Args: json.RawMessage(buf),
	})
	if err := config.SaveConfigToPath(files.ProvConfig, &provConfig); err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}
