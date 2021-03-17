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
	"github.com/google/go-cmp/cmp"
	"github.com/google/subcommands"
	compute "google.golang.org/api/compute/v1"
)

func executeInstallGPU(ctx context.Context, files *fs.Files, gcs *storage.Client, flgs ...string) (subcommands.ExitStatus, error) {
	clients := ServiceClients(func(context.Context, bool) (*compute.Service, *storage.Client, error) {
		return nil, gcs, nil
	})
	fs := &flag.FlagSet{}
	installGPU := &InstallGPU{}
	installGPU.SetFlags(fs)
	if err := fs.Parse(flgs); err != nil {
		return 0, err
	}
	ret := installGPU.Execute(ctx, fs, files, clients)
	if ret != subcommands.ExitSuccess {
		return ret, fmt.Errorf("InstallGPU failed. input: %v", flgs)
	}
	return ret, nil
}

func TestGetValidDriverVersions(t *testing.T) {
	testData := []struct {
		testName string
		objects  map[string][]byte
		want     map[string]bool
	}{
		{
			"NonEmpty",
			map[string][]byte{
				"/nvidia-drivers-us-public/tesla/396.26/obj-1": nil,
				"/nvidia-drivers-us-public/tesla/396.44/obj-2": nil,
			},
			map[string]bool{"390.46": true, "396.26": true, "396.44": true},
		},
		{
			"Empty",
			nil,
			map[string]bool{"390.46": true},
		},
	}
	gcs := fakes.GCSForTest(t)
	defer gcs.Close()
	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			gcs.Objects = input.objects
			got, err := validDriverVersions(context.Background(), gcs.Client)
			if err != nil {
				t.Fatal(err)
			}
			if !cmp.Equal(got, input.want) {
				t.Errorf("validDriverVersions; got %v, want %v; objects:\n%v", got, input.want, input.objects)
			}
		})
	}
}

func TestGetValidDriverVersionsNoOp(t *testing.T) {
	gcs := fakes.GCSForTest(t)
	defer gcs.Close()
	if _, err := executeInstallGPU(context.Background(), nil, gcs.Client, "-get-valid-drivers"); err != nil {
		t.Fatalf("install-gpu(-get-valid-drivers); failed with nil files input; err %q; should succeed", err)
	}
}

func setupInstallGPUFiles() (string, *fs.Files, error) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", nil, err
	}
	files := &fs.Files{}
	files.ProvConfig, err = createTempFile(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", nil, err
	}
	if err := ioutil.WriteFile(files.ProvConfig, []byte("{}"), 0644); err != nil {
		os.RemoveAll(tmpDir)
		return "", nil, err
	}
	buildConfigFile, err := ioutil.TempFile(tmpDir, "")
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", nil, err
	}
	if err := config.Save(buildConfigFile, struct{}{}); err != nil {
		buildConfigFile.Close()
		os.RemoveAll(tmpDir)
		return "", nil, err
	}
	if err := buildConfigFile.Close(); err != nil {
		os.RemoveAll(tmpDir)
		return "", nil, err
	}
	files.BuildConfig = buildConfigFile.Name()
	return tmpDir, files, nil
}

func TestInstallGPUBuildConfig(t *testing.T) {
	tmpDir, files, err := setupInstallGPUFiles()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	gcs := fakes.GCSForTest(t)
	defer gcs.Close()
	if _, err := executeInstallGPU(context.Background(), files, gcs.Client, "-version=390.46", "-gpu-type=nvidia-tesla-k80"); err != nil {
		t.Fatal(err)
	}
	buildConfig := &config.Build{}
	if err := config.LoadFromFile(files.BuildConfig, buildConfig); err != nil {
		t.Fatal(err)
	}
	if got := buildConfig.GPUType; got != "nvidia-tesla-k80" {
		t.Errorf("install-gpu(-version=390.46 -gpu-type=nvidia-tesla-k80); GPU; got %s, want nvidia-tesla-k80", buildConfig.GPUType)
	}
}

func TestInstallGPUBuildConfigGCSFiles(t *testing.T) {
	tmpDir, files, err := setupInstallGPUFiles()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	depsDir := filepath.Join(tmpDir, "deps")
	if err := os.Mkdir(depsDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(filepath.Join(depsDir, "test-file"), []byte("test-file"), 0644); err != nil {
		t.Fatal(err)
	}
	gcs := fakes.GCSForTest(t)
	defer gcs.Close()
	if _, err := executeInstallGPU(context.Background(), files, gcs.Client, "-version=390.46", "-deps-dir="+depsDir); err != nil {
		t.Fatal(err)
	}
	buildConfig := &config.Build{}
	if err := config.LoadFromFile(files.BuildConfig, buildConfig); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(depsDir, "test-file")
	foundWant := false
	for _, got := range buildConfig.GCSFiles {
		if got == want {
			foundWant = true
			break
		}
	}
	if !foundWant {
		t.Errorf("install-gpu(-version=390.46 -deps-dir=%q); buildConfig.GCSFiles; got %v, must include %q", depsDir, buildConfig.GCSFiles, want)
	}
}

func TestInstallGPUInvalidVersion(t *testing.T) {
	tmpDir, files, err := setupInstallGPUFiles()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	gcs := fakes.GCSForTest(t)
	defer gcs.Close()
	if _, err := executeInstallGPU(context.Background(), files, gcs.Client, "-version=bad"); err == nil {
		t.Error("install-gpu(-version=bad); got nil, want error")
	}
}

func TestInstallGPUInvalidGPUType(t *testing.T) {
	tmpDir, files, err := setupInstallGPUFiles()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	gcs := fakes.GCSForTest(t)
	defer gcs.Close()
	if _, err := executeInstallGPU(context.Background(), files, gcs.Client, "-version=390.46", "-gpu-type=bad"); err == nil {
		t.Error("install-gpu(-version=390.46 -gpu-type=bad); got nil, want error")
	}
}

func TestInstallGPURunTwice(t *testing.T) {
	tmpDir, files, err := setupInstallGPUFiles()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	gcs := fakes.GCSForTest(t)
	defer gcs.Close()
	if _, err := executeInstallGPU(context.Background(), files, gcs.Client, "-version=390.46"); err != nil {
		t.Fatal(err)
	}
	if _, err = executeInstallGPU(context.Background(), files, gcs.Client, "-version=390.46"); err == nil {
		t.Error("install-gpu(_); run twice; got nil, want error")
	}
}

func TestInstallGPUProvisionerConfig(t *testing.T) {
	tmpDir, files, err := setupInstallGPUFiles()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	gcs := fakes.GCSForTest(t)
	defer gcs.Close()
	if _, err := executeInstallGPU(context.Background(), files, gcs.Client, "-version=390.46"); err != nil {
		t.Fatal(err)
	}
	want := provisioner.Config{
		Steps: []provisioner.StepConfig{
			{
				Type: "InstallGPU",
				Args: mustMarshalJSON(t, &provisioner.InstallGPUStep{
					NvidiaDriverVersion:      "390.46",
					NvidiaInstallDirHost:     "/var/lib/nvidia",
					NvidiaInstallerContainer: installerContainer,
				}),
			},
		},
	}
	var got provisioner.Config
	data, err := ioutil.ReadFile(files.ProvConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("install-gpu(-version=390.46); provisioner config mismatch; diff (-got, +want): %s", diff)
	}
}

func TestInstallGPUInstallerWithoutDepsDir(t *testing.T) {
	tmpDir, files, err := setupInstallGPUFiles()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	gcs := fakes.GCSForTest(t)
	defer gcs.Close()
	if _, err := executeInstallGPU(context.Background(), files, gcs.Client, "-version=NVIDIA-Linux-x86_64-450.51.06.run"); err == nil {
		t.Error("install-gpu(-version=NVIDIA-Linux-x86_64-450.51.06.run); got nil, want error")
	}
}
