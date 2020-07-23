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
	"os"
	"path/filepath"
	"testing"

	"cos-customizer/config"
	"cos-customizer/fakes"
	"cos-customizer/fs"

	"cloud.google.com/go/storage"
	"github.com/google/subcommands"
	compute "google.golang.org/api/compute/v1"
)

func executeFinishBuild(files *fs.Files, svc *compute.Service, gcs *storage.Client, flags ...string) (subcommands.ExitStatus, error) {
	clients := ServiceClients(func(_ context.Context, _ bool) (*compute.Service, *storage.Client, error) {
		return svc, gcs, nil
	})
	flagSet := &flag.FlagSet{}
	finishBuild := &FinishImageBuild{}
	finishBuild.SetFlags(flagSet)
	if err := flagSet.Parse(flags); err != nil {
		return 0, err
	}
	ret := finishBuild.Execute(context.Background(), flagSet, files, clients)
	if ret != subcommands.ExitSuccess {
		return ret, fmt.Errorf("FinishImageBuild failed; input: %v", flags)
	}
	return ret, nil
}

func setupFinishBuildFiles() (string, *fs.Files, error) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", nil, err
	}
	files := &fs.Files{}
	files.PersistBuiltinBuildContext, err = ioutil.TempDir(tmpDir, "")
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", nil, err
	}
	files.BuiltinBuildContextArchive = filepath.Join(tmpDir, "builtin_archive")
	files.UserBuildContextArchive, err = createTempFile(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", nil, err
	}
	files.StateFile, err = createTempFile(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", nil, err
	}
	files.StartupScript, err = createTempFile(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", nil, err
	}
	files.SystemdService, err = createTempFile(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", nil, err
	}
	files.DaisyWorkflow, err = createTempFile(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", nil, err
	}
	buildConfigFile, err := ioutil.TempFile(tmpDir, "")
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", nil, err
	}
	if err := config.Save(buildConfigFile, &config.Build{GCSBucket: "b", GCSDir: "d"}); err != nil {
		buildConfigFile.Close()
		os.RemoveAll(tmpDir)
		return "", nil, err
	}
	if err := buildConfigFile.Close(); err != nil {
		os.RemoveAll(tmpDir)
		return "", nil, err
	}
	files.BuildConfig = buildConfigFile.Name()
	sourceImageFile, err := ioutil.TempFile(tmpDir, "")
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", nil, err
	}
	if err := config.Save(sourceImageFile, config.NewImage("in", "p")); err != nil {
		sourceImageFile.Close()
		os.RemoveAll(tmpDir)
		return "", nil, err
	}
	if err := sourceImageFile.Close(); err != nil {
		os.RemoveAll(tmpDir)
		return "", nil, err
	}
	files.SourceImageConfig = sourceImageFile.Name()
	files.DaisyBin = "/bin/true"
	return tmpDir, files, nil
}

func TestOutputImageExists(t *testing.T) {
	tmpDir, files, err := setupFinishBuildFiles()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	gcs := fakes.GCSForTest(t)
	gce, svc := fakes.GCEForTest(t, "p")
	gce.Images = &compute.ImageList{Items: []*compute.Image{{Name: "out"}}}
	files.DaisyBin = "/bin/false"
	if _, err := executeFinishBuild(files, svc, gcs.Client, "-project=p", "-zone=z", "-image-name=out", "-image-project=p"); err != nil {
		t.Logf("images: %v", gce.Images)
		t.Errorf("FinishImageBuild.Execute(-image-name=out -image-project=p); daisy shouldn't execute if image exists; err: %q", err)
	}
}

func TestOutputImageSuffixExists(t *testing.T) {
	tmpDir, files, err := setupFinishBuildFiles()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	gcs := fakes.GCSForTest(t)
	gce, svc := fakes.GCEForTest(t, "p")
	gce.Images = &compute.ImageList{Items: []*compute.Image{{Name: "in-out"}}}
	files.DaisyBin = "/bin/false"
	if _, err := executeFinishBuild(files, svc, gcs.Client, "-project=p", "-zone=z", "-image-suffix=-out", "-image-project=p"); err != nil {
		t.Logf("images: %v", gce.Images)
		t.Errorf("FinishImageBuild.Execute(-image-suffix=-out -image-project=p); daisy shouldn't execute if image exists; err: %q", err)
	}
}

func TestDeprecateImages(t *testing.T) {
	tmpDir, files, err := setupFinishBuildFiles()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	gcs := fakes.GCSForTest(t)
	gce, svc := fakes.GCEForTest(t, "p")
	gce.Images = &compute.ImageList{Items: []*compute.Image{{Name: "old", Family: "f"}}}
	gce.Operations = []*compute.Operation{{Status: "DONE"}}
	if _, err := executeFinishBuild(files, svc, gcs.Client, "-project=p", "-zone=z", "-image-name=out", "-image-project=p", "-image-family=f", "-deprecate-old-images"); err != nil {
		t.Fatal(err)
	}
	if _, ok := gce.Deprecated["old"]; !ok {
		t.Errorf("Image 'old' is not deprecated; deprecated images: %v", gce.Deprecated)
	}
}

func TestValidateFailure(t *testing.T) {
	tests := []struct {
		name      string
		flags     []string
		expectErr bool
		msg       string
	}{
		{
			name:      "Timeout",
			flags:     []string{"-project=p", "-zone=z", "-image-name=out", "-image-project=p", "-image-family=f", "-timeout=t"},
			expectErr: true,
			msg:       "'timeout' value should be invalid",
		}, {
			name:      "SmallDiskSize",
			flags:     []string{"-project=p", "-zone=z", "-image-name=out", "-image-project=p", "-image-family=f", "-disk-size-gb=12", "-oem-size=1025M"},
			expectErr: true,
			msg:       "disk size should be invalid",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tmpDir, files, err := setupFinishBuildFiles()
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)
			gcs := fakes.GCSForTest(t)
			_, svc := fakes.GCEForTest(t, "p")
			if _, err := executeFinishBuild(files, svc, gcs.Client, test.flags...); test.expectErr && err == nil {
				t.Errorf("Got nil, want error; %s", test.msg)
			}
		})
	}
}
