// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/config"
	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/fakes"
	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/fs"
	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/provisioner"

	"cloud.google.com/go/storage"
	"github.com/google/subcommands"
	compute "google.golang.org/api/compute/v1"
)

func executeStartBuild(files *fs.Files, svc *compute.Service, flags ...string) (subcommands.ExitStatus, error) {
	clients := ServiceClients(func(_ context.Context, _ bool) (*compute.Service, *storage.Client, error) {
		return svc, nil, nil
	})
	flagSet := &flag.FlagSet{}
	startBuild := &StartImageBuild{}
	startBuild.SetFlags(flagSet)
	if err := flagSet.Parse(flags); err != nil {
		return 0, err
	}
	ret := startBuild.Execute(context.Background(), flagSet, files, clients)
	if ret != subcommands.ExitSuccess {
		return ret, fmt.Errorf("StartImageBuild failed; input: %v", flags)
	}
	return ret, nil
}

func setupStartBuildFiles() (*fs.Files, string, error) {
	files := &fs.Files{}
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, "", err
	}
	files.BuildConfig = filepath.Join(tmpDir, "build_config")
	files.SourceImageConfig = filepath.Join(tmpDir, "source_image")
	files.ProvConfig = filepath.Join(tmpDir, "provisioner_config")
	files.UserBuildContextArchive = filepath.Join(tmpDir, "user_archive")
	return files, tmpDir, nil
}

func TestNoImageName(t *testing.T) {
	files, tmpDir, err := setupStartBuildFiles()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	gce, client := fakes.GCEForTest(t, "p")
	defer gce.Close()
	if _, err := executeStartBuild(files, client, "-gcs-bucket=b", "-gcs-workdir=w", "-image-project=p"); err == nil {
		t.Errorf("start-image-build should fail with no image name")
	}
}

func TestDuplicateImageName(t *testing.T) {
	files, tmpDir, err := setupStartBuildFiles()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	gce, client := fakes.GCEForTest(t, "p")
	defer gce.Close()
	if _, err := executeStartBuild(files, client, "-image-name=n", "-image-family=f", "-gcs-bucket=b", "-gcs-workdir=w",
		"-image-project=p"); err == nil {
		t.Errorf("start-image-build should fail with duplicate image names")
	}
}

func TestResolveMilestoneNoCosCloud(t *testing.T) {
	files, tmpDir, err := setupStartBuildFiles()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	gce, client := fakes.GCEForTest(t, "p")
	defer gce.Close()
	if _, err := executeStartBuild(files, client, "-image-milestone=65", "-image-project=p", "-gcs-bucket=b",
		"-gcs-workdir=w"); err == nil {
		t.Errorf("start-image-build should fail when using -image-milestone without cos-cloud")
	}
}

func TestSourceImage(t *testing.T) {
	testData := []struct {
		testName string
		images   []*compute.Image
		flag     string
		want     string
	}{
		{
			"MilestoneDifferentImages",
			[]*compute.Image{
				{Name: "cos-beta-65-10032-9-0"},
				{Name: "cos-stable-65-10032-10-0"}},
			"-image-milestone=65",
			"cos-stable-65-10032-10-0",
		},
		{
			"MilestoneSameImage",
			[]*compute.Image{
				{Name: "cos-stable-65-10032-10-0"},
				{Name: "cos-65-10032-10-0"}},
			"-image-milestone=65",
			"cos-stable-65-10032-10-0",
		},
		{
			"ProvideImageName",
			[]*compute.Image{
				{Name: "cos-beta-65-10032-9-0"},
				{Name: "cos-stable-65-10032-10-0"}},
			"-image-name=cos-beta-65-10032-9-0",
			"cos-beta-65-10032-9-0",
		},
	}
	gce, client := fakes.GCEForTest(t, "cos-cloud")
	defer gce.Close()
	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			files, tmpDir, err := setupStartBuildFiles()
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)
			gce.Images.Items = input.images
			if _, err := executeStartBuild(files, client, input.flag, "-image-project=cos-cloud", "-gcs-bucket=b",
				"-gcs-workdir=w"); err != nil {
				t.Fatal(err)
			}
			sourceImage := config.NewImage("", "")
			if err := config.LoadFromFile(files.SourceImageConfig, sourceImage); err != nil {
				t.Fatal(err)
			}
			if got := sourceImage.Name; got != input.want {
				t.Errorf("StartImageBuild.Execute(%s); source image is %s, want %s", input.flag, got, input.want)
			}
			if got := sourceImage.Project; got != "cos-cloud" {
				t.Errorf("StartImageBuild.Execute(%s); source image project is %s, want cos-cloud", input.flag, got)
			}
		})
	}
}

func TestProvisionerConfigCreated(t *testing.T) {
	files, tmpDir, err := setupStartBuildFiles()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	gce, client := fakes.GCEForTest(t, "p")
	defer gce.Close()
	gce.Images.Items = []*compute.Image{{Name: "n"}}
	if _, err := executeStartBuild(files, client, "-image-name=n", "-image-project=p", "-gcs-bucket=b", "-gcs-workdir=w"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(files.ProvConfig); os.IsNotExist(err) {
		t.Errorf("provisioner config does not exist: should exist")
	}
	var got provisioner.Config
	data, err := ioutil.ReadFile(files.ProvConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Errorf("cannot unmarshal provisioner config %q: got %v", string(data), err)
	}
}
