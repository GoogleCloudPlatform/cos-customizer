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

package cmd

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"cos-customizer/fs"

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

func quoteForShell(str string) string {
	return fmt.Sprintf("'%s'", strings.Replace(str, "'", "'\"'\"'", -1))
}

// createEnvFile creates an environment variable file from the given map. During preloading, this file
// is sourced before the script associated with this step is run. The resulting file is stored in
// the builtin build context to avoid collisions with user data.
func createEnvFile(prefix string, files *fs.Files, env map[string]string) (string, error) {
	if env == nil || len(env) == 0 {
		return "", nil
	}
	envFile, err := ioutil.TempFile(files.PersistBuiltinBuildContext, prefix)
	if err != nil {
		return "", err
	}
	for k, v := range env {
		if _, err := fmt.Fprintf(envFile, "export %s=%s\n", k, quoteForShell(v)); err != nil {
			envFile.Close()
			os.Remove(envFile.Name())
			return "", err
		}
	}
	if err := envFile.Close(); err != nil {
		os.Remove(envFile.Name())
		return "", err
	}
	return filepath.Base(envFile.Name()), nil
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
	envFileName, err := createEnvFile("user_env_", files, r.env.m)
	if err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}
	if err := fs.AppendStateFile(files.StateFile, fs.User, r.script, envFileName); err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}
