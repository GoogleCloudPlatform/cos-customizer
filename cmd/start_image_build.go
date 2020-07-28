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

// Package cmd contains cos-customizer subcommand implementations.
package cmd

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"cos-customizer/config"
	"cos-customizer/fs"
	"cos-customizer/gce"

	"cloud.google.com/go/storage"
	"github.com/google/subcommands"
	compute "google.golang.org/api/compute/v1"
)

// ServiceClients gets the GCE and GCS clients to use.
type ServiceClients func(ctx context.Context, anonymousCreds bool) (*compute.Service, *storage.Client, error)

// StartImageBuild implements subcommands.Command for the 'start-image-build' command.
// This command initializes a new image customization process.
type StartImageBuild struct {
	buildContext string
	gcsBucket    string
	gcsWorkdir   string
	imageProject string
	imageName    string
	milestone    int
	imageFamily  string
}

// Name implements subcommands.Command.Name.
func (*StartImageBuild) Name() string {
	return "start-image-build"
}

// Synopsis implements subcommands.Command.Synopsis.
func (*StartImageBuild) Synopsis() string {
	return "Start a COS image build."
}

// Usage implements subcommands.Command.Usage.
func (*StartImageBuild) Usage() string {
	return `start-image-build [flags]
`
}

// SetFlags implements subcommands.Command.SetFlags.
func (s *StartImageBuild) SetFlags(f *flag.FlagSet) {
	f.StringVar(&s.buildContext, "build-context", ".", "Path to the build context")
	f.StringVar(&s.gcsBucket, "gcs-bucket", "", "GCS bucket to use for scratch space")
	f.StringVar(&s.gcsWorkdir, "gcs-workdir", "", "GCS directory to use for scratch space")
	f.StringVar(&s.imageProject, "image-project", "", "Source image project")
	f.StringVar(&s.imageName, "image-name", "", "Source image name. Mutually exclusive with 'image-milestone' and "+
		"'image-family'.")
	f.IntVar(&s.milestone, "image-milestone", 0, "Source image milestone. Mutually exclusive with 'image-name' "+
		"and 'image-family'. Can only be used if 'image-project' is cos-cloud.")
	f.StringVar(&s.imageFamily, "image-family", "", "Source image family. Mutually exclusive with 'image-name' "+
		"and 'image-milestone'.")
}

func (s *StartImageBuild) validate() error {
	numSet := 0
	for _, val := range []bool{s.imageName != "", s.milestone != 0, s.imageFamily != ""} {
		if val {
			numSet++
		}
	}
	switch {
	case numSet != 1:
		return fmt.Errorf("exactly one of image-name, image-milestone, image-family must be set")
	case s.milestone != 0 && s.imageProject != "cos-cloud":
		return fmt.Errorf("image-milestone can only be used if image-project is set to cos-cloud. "+
			"image-milestone: %d image-project: %s", s.milestone, s.imageProject)
	case s.gcsBucket == "":
		return fmt.Errorf("gcs-bucket must be set")
	case s.gcsWorkdir == "":
		return fmt.Errorf("gcs-workdir must be set")
	case s.imageProject == "":
		return fmt.Errorf("image-project must be set")
	default:
		return nil
	}
}

func (s *StartImageBuild) resolveImageName(ctx context.Context, svc *compute.Service) error {
	switch {
	case s.milestone != 0:
		var err error
		s.imageName, err = gce.ResolveMilestone(ctx, svc, s.milestone)
		if err != nil {
			if err == gce.ErrImageNotFound {
				return fmt.Errorf("no image found on milestone %d", s.milestone)
			}
			return err
		}
		log.Printf("Using image %s from milestone %d\n", s.imageName, s.milestone)
	case s.imageFamily != "":
		image, err := svc.Images.GetFromFamily(s.imageProject, s.imageFamily).Do()
		if err != nil {
			return err
		}
		s.imageName = image.Name
		log.Printf("Using image %s from family %s\n", s.imageName, s.imageFamily)
	default:
		exists, err := gce.ImageExists(svc, s.imageProject, s.imageName)
		if err != nil {
			return err
		}
		if !exists {
			return fmt.Errorf("could not find source image %s in project %s", s.imageName, s.imageProject)
		}
	}
	return nil
}

func saveImage(imageName, imageProject, dst string) error {
	image := config.NewImage(imageName, imageProject)
	if err := os.MkdirAll(filepath.Dir(dst), 0774); err != nil {
		return err
	}
	outFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer outFile.Close()
	return config.Save(outFile, image)
}

func saveBuildConfig(gcsBucket, gcsWorkdir, dst string) error {
	buildConfig := &config.Build{GCSBucket: gcsBucket, GCSDir: gcsWorkdir}
	if err := os.MkdirAll(filepath.Dir(dst), 0774); err != nil {
		return err
	}
	outFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer outFile.Close()
	return config.SaveBuildConfigToFile(outFile, buildConfig)
}

// Execute implements subcommands.Command.Execute. It initializes persistent state for a new
// image customization process.
func (s *StartImageBuild) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	if f.NArg() != 0 {
		f.Usage()
		return subcommands.ExitUsageError
	}
	files := args[0].(*fs.Files)
	svc, _, err := args[1].(ServiceClients)(ctx, false)
	if err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}
	if err := s.validate(); err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}
	if err := s.resolveImageName(ctx, svc); err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}
	if err := fs.CreateBuildContextArchive(s.buildContext, files.UserBuildContextArchive); err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}
	if err := fs.CreateStateFile(files); err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}
	if err := fs.CreatePersistentBuiltinContext(files); err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}
	if err := saveImage(s.imageName, s.imageProject, files.SourceImageConfig); err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}
	if err := saveBuildConfig(s.gcsBucket, s.gcsWorkdir, files.BuildConfig); err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}
	return subcommands.ExitSuccess
}
