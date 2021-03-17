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
	userBuildContextArchive = "user_build_context.tar"
	sourceImageConfig       = "config/source_image"
	buildConfig             = "config/build"
	provConfig              = "config/provisioner"

	// Volatile files. These paths exist in the volatileDir at container start time.
	// Changes to these files do not persist across build steps.
	daisyWorkflow = "build_image.wf.json"
)

// Files stores important file paths.
type Files struct {
	persistentDir string
	// UserBuildContextArchive points to the tar archive of the user build context.
	// The user build context contains user provided scripts and files that users can use during preloading.
	UserBuildContextArchive string
	// SourceImageConfig points to the source image configuration.
	SourceImageConfig string
	// BuildConfig points to the image build process configuration.
	BuildConfig string
	// ProvConfig points to the provisioner configuration that runs on the preload
	// VM.
	ProvConfig string
	// DaisyWorkflow points to the Daisy workflow to template and use for preloading.
	DaisyWorkflow string
	// DaisyBin points to the Daisy binary.
	DaisyBin string
}

// DefaultFiles builds a Files struct with a default file layout.
func DefaultFiles(persistentDir string) *Files {
	persistentDir = filepath.Join(os.Getenv("HOME"), persistentDir)
	return &Files{
		persistentDir:           persistentDir,
		UserBuildContextArchive: filepath.Join(persistentDir, userBuildContextArchive),
		SourceImageConfig:       filepath.Join(persistentDir, sourceImageConfig),
		BuildConfig:             filepath.Join(persistentDir, buildConfig),
		ProvConfig:              filepath.Join(persistentDir, provConfig),
		DaisyWorkflow:           filepath.Join(volatileDir, daisyWorkflow),
		DaisyBin:                daisyBin,
	}
}

// CleanupAllPersistent deletes everything in the persistent directory.
func (f *Files) CleanupAllPersistent() error {
	return os.RemoveAll(f.persistentDir)
}
