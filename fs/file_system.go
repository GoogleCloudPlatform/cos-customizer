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

// Package fs exports functionality related to all of the cos-customizer
// state stored on the file system.
package fs

import (
	"os"
	"path/filepath"
)

const (
	// ScratchDir is used for temp files and the like.
	ScratchDir = "/tmp"

	// DaisyBin is the location of the Daisy binary.
	daisyBin = "/daisy"

	// Directory whose contents do not persist across build steps.
	// This directory is used for building files into the container image.
	volatileDir = "/data"

	// Persistent files. These paths need to be created before they are used.
	// Changes to these files persist across build steps.
	userBuildContextArchive    = "user_build_context.tar"
	builtinBuildContextArchive = "builtin_build_context.tar"
	stateFile                  = "state_file"
	sourceImageConfig          = "config/source_image"
	buildConfig                = "config/build"

	// Volatile files. These paths exist in the volatileDir at container start time.
	// Changes to these files do not persist across build steps.
	daisyWorkflow  = "build_image.wf.json"
	startupScript  = "startup.sh"
	systemdService = "customizer.service"

	// Persistent and volatile files. These paths exist in both the persistentDir and volatileDir,
	// but are expected to be used from the persistentDir.
	builtinBuildContext = "builtin_build_context"
)

// Files stores important file paths.
type Files struct {
	persistentDir string
	// UserBuildContextArchive points to the tar archive of the user build context.
	// The user build context contains user provided scripts and files that users can use during preloading.
	UserBuildContextArchive string
	// BuiltinBuildContextArchive points to the tar archive of the builtin build context.
	// The builtin build context contains scripts and files used by cos-customizer build steps.
	BuiltinBuildContextArchive string
	// PersistBuiltinBuildContext points to the directory containing the persistent builtin build context.
	// The persistent builtin build context persists across build steps.
	PersistBuiltinBuildContext string
	// StateFile points to the state file. This file encodes the sequence of instructions that need
	// to be executed on the preload VM.
	StateFile string
	// SourceImageConfig points to the source image configuration.
	SourceImageConfig string
	// BuildConfig points to the image build process configuration.
	BuildConfig string
	// DaisyWorkflow points to the Daisy workflow to template and use for preloading.
	DaisyWorkflow string
	// StartupScript points to the startup script that needs to run on the preload VM.
	StartupScript string
	// SystemdService points to the systemd service that needs to invoke the startup script on the preload VM.
	SystemdService string
	// VolatileBuiltinBuildContext points to the directory containing the volatile builtin build context.
	// The volatile builtin build context does not persist across build steps; changes made by one
	// build step are not seen by other build steps.
	VolatileBuiltinBuildContext string
	// DaisyBin points to the Daisy binary.
	DaisyBin string
}

// DefaultFiles builds a Files struct with a default file layout.
func DefaultFiles(persistentDir string) *Files {
	persistentDir = filepath.Join(os.Getenv("HOME"), persistentDir)
	return &Files{
		persistentDir,
		filepath.Join(persistentDir, userBuildContextArchive),
		filepath.Join(persistentDir, builtinBuildContextArchive),
		filepath.Join(persistentDir, builtinBuildContext),
		filepath.Join(persistentDir, stateFile),
		filepath.Join(persistentDir, sourceImageConfig),
		filepath.Join(persistentDir, buildConfig),
		filepath.Join(volatileDir, daisyWorkflow),
		filepath.Join(volatileDir, startupScript),
		filepath.Join(volatileDir, systemdService),
		filepath.Join(volatileDir, builtinBuildContext),
		daisyBin,
	}
}

// CleanupAllPersistent deletes everything in the persistent directory.
func (f *Files) CleanupAllPersistent() error {
	return os.RemoveAll(f.persistentDir)
}
