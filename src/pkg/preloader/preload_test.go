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

package preloader

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/config"
	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/fakes"
	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/fs"
	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/provisioner"

	"github.com/google/go-cmp/cmp"
	compute "google.golang.org/api/compute/v1"
)

func createTempFile(dir string) (string, error) {
	file, err := ioutil.TempFile(dir, "")
	if err != nil {
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return file.Name(), nil
}

func setupFiles() (string, *fs.Files, error) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", nil, err
	}
	files := &fs.Files{}
	files.UserBuildContextArchive, err = createTempFile(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", nil, err
	}
	files.ProvConfig, err = createTempFile(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", nil, err
	}
	files.DaisyWorkflow, err = createTempFile(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", nil, err
	}
	return tmpDir, files, nil
}

func TestDaisyArgsGCSUpload(t *testing.T) {
	tmpDir, files, err := setupFiles()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	if err := ioutil.WriteFile(filepath.Join(tmpDir, "test-file"), []byte("test-file"), 0644); err != nil {
		t.Fatal(err)
	}
	var testData = []struct {
		testName string
		file     string
		object   string
		contents []byte
	}{
		{
			testName: "UserBuildContextArchive",
			file:     files.UserBuildContextArchive,
			object:   filepath.Base(files.UserBuildContextArchive),
			contents: []byte("abc"),
		},
		{
			testName: "ArbitraryFileUpload",
			file:     filepath.Join(tmpDir, "test-file"),
			object:   "gcs_files/test-file",
			contents: []byte("test-file"),
		},
	}
	gcs := fakes.GCSForTest(t)
	defer gcs.Close()
	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			gcs.Objects = make(map[string][]byte)
			gm := &gcsManager{gcsClient: gcs.Client, gcsBucket: "bucket"}
			if err := ioutil.WriteFile(input.file, input.contents, 0744); err != nil {
				t.Fatal(err)
			}
			buildSpec := &config.Build{
				GCSFiles: []string{filepath.Join(tmpDir, "test-file")},
			}
			if _, err := daisyArgs(context.Background(), gm, files, config.NewImage("", ""), config.NewImage("", ""), buildSpec, &provisioner.Config{}); err != nil {
				t.Fatalf("daisyArgs: %v", err)
			}
			got, ok := gcs.Objects[fmt.Sprintf("/bucket/cos-customizer/%s", input.object)]
			if !ok {
				t.Fatalf("daisyArgs: write /bucket/cos-customizer/%s: not found", input.object)
			}
			if !cmp.Equal(got, input.contents) {
				t.Errorf("daisyArgs: write /bucket/cos-customizer/%s: got %s, want %s", input.object, string(got), string(input.contents))
			}
		})
	}
}

func getDaisyVarValue(variable string, args []string) (string, bool) {
	for i, arg := range args {
		if arg == fmt.Sprintf("-var:%s", variable) {
			return args[i+1], true
		}
	}
	return "", false
}

func TestDaisyArgsWorkflowTemplate(t *testing.T) {
	var testData = []struct {
		testName    string
		outputImage *config.Image
		buildConfig *config.Build
		workflow    []byte
		want        []byte
	}{
		{
			testName:    "Empty",
			outputImage: config.NewImage("", ""),
			buildConfig: &config.Build{GCSBucket: "bucket"},
			workflow:    []byte("{{.Licenses}} {{.Labels}} {{.Accelerators}}"),
			want:        []byte("null {} []"),
		},
		{
			testName:    "OneLicense",
			outputImage: &config.Image{Image: &compute.Image{Licenses: []string{"my-license"}}, Project: ""},
			buildConfig: &config.Build{GCSBucket: "bucket"},
			workflow:    []byte("{{.Licenses}}"),
			want:        []byte("[\"my-license\"]"),
		},
		{
			testName:    "TwoLicenses",
			outputImage: &config.Image{Image: &compute.Image{Licenses: []string{"license-1", "license-2"}}, Project: ""},
			buildConfig: &config.Build{GCSBucket: "bucket"},
			workflow:    []byte("{{.Licenses}}"),
			want:        []byte("[\"license-1\",\"license-2\"]"),
		},
		{
			testName:    "EmptyStringLicense",
			outputImage: &config.Image{Image: &compute.Image{Licenses: []string{""}}, Project: ""},
			buildConfig: &config.Build{GCSBucket: "bucket"},
			workflow:    []byte("{{.Licenses}}"),
			want:        []byte("null"),
		},
		{
			testName:    "OneEmptyLicense",
			outputImage: &config.Image{Image: &compute.Image{Licenses: []string{"license-1", ""}}, Project: ""},
			buildConfig: &config.Build{GCSBucket: "bucket"},
			workflow:    []byte("{{.Licenses}}"),
			want:        []byte("[\"license-1\"]"),
		},
		{
			testName:    "URLLicense",
			outputImage: &config.Image{Image: &compute.Image{Licenses: []string{"https://www.googleapis.com/compute/v1/projects/my-proj/global/licenses/my-license"}}, Project: ""},
			buildConfig: &config.Build{GCSBucket: "bucket"},
			workflow:    []byte("{{.Licenses}}"),
			want:        []byte("[\"projects/my-proj/global/licenses/my-license\"]"),
		},
		{
			testName:    "Labels",
			outputImage: &config.Image{Image: &compute.Image{Labels: map[string]string{"key": "value"}}, Project: ""},
			buildConfig: &config.Build{GCSBucket: "bucket"},
			workflow:    []byte("{{.Labels}}"),
			want:        []byte("{\"key\":\"value\"}"),
		},
		{
			testName:    "Accelerators",
			outputImage: config.NewImage("", ""),
			buildConfig: &config.Build{GCSBucket: "bucket", GPUType: "nvidia-tesla-k80", Project: "p", Zone: "z"},
			workflow:    []byte("{{.Accelerators}}"),
			want:        []byte("[{\"acceleratorCount\":1,\"acceleratorType\":\"projects/p/zones/z/acceleratorTypes/nvidia-tesla-k80\"}]"),
		},
	}
	gcs := fakes.GCSForTest(t)
	defer gcs.Close()
	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			tmpDir, files, err := setupFiles()
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)
			gcs.Objects = make(map[string][]byte)
			gm := &gcsManager{gcs.Client, input.buildConfig.GCSBucket, input.buildConfig.GCSDir}
			if err := ioutil.WriteFile(files.DaisyWorkflow, input.workflow, 0744); err != nil {
				t.Fatal(err)
			}
			args, err := daisyArgs(context.Background(), gm, files, config.NewImage("", ""), input.outputImage, input.buildConfig, &provisioner.Config{})
			if err != nil {
				t.Fatalf("daisyArgs: %v", err)
			}
			got, err := ioutil.ReadFile(args[len(args)-1])
			if err != nil {
				t.Fatal(err)
			}
			if !cmp.Equal(got, input.want) {
				t.Errorf("daisyArgs: template Daisy: got %s, want %s", string(got), string(input.want))
			}
		})
	}
}

func isSubSlice(a, b []string) bool {
	switch {
	case a == nil || len(a) == 0:
		return true
	case b == nil || len(a) > len(b):
		return false
	}
	for i := len(a); i <= len(b); i++ {
		subslice := b[i-len(a) : i]
		if cmp.Equal(a, subslice) {
			return true
		}
	}
	return false
}

func mustMarshalJSON(t *testing.T, v interface{}) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestDaisyArgs(t *testing.T) {
	tmpDir, files, err := setupFiles()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	var testData = []struct {
		testName          string
		inputImage        *config.Image
		outputImage       *config.Image
		buildConfig       *config.Build
		provConfig        *provisioner.Config
		want              []string
		wantBuildContexts map[string]string
		wantSteps         []provisioner.StepConfig
		wantBootDisk      *provisioner.BootDiskConfig
	}{
		{
			testName:    "GPU",
			inputImage:  config.NewImage("", ""),
			outputImage: config.NewImage("", ""),
			buildConfig: &config.Build{GPUType: "nvidia-tesla-k80", GCSBucket: "bucket", GCSDir: "dir"},
			want:        []string{"-var:host_maintenance", "TERMINATE"},
		},
		{
			testName:    "NoGPU",
			inputImage:  config.NewImage("", ""),
			outputImage: config.NewImage("", ""),
			buildConfig: &config.Build{GCSBucket: "bucket", GCSDir: "dir"},
			want:        []string{"-var:host_maintenance", "MIGRATE"},
		},
		{
			testName:    "SourceImage",
			inputImage:  config.NewImage("im", "proj"),
			outputImage: config.NewImage("", ""),
			buildConfig: &config.Build{GCSBucket: "bucket", GCSDir: "dir"},
			want:        []string{"-var:source_image", "projects/proj/global/images/im"},
		},
		{
			testName:    "OutputImageName",
			inputImage:  config.NewImage("", ""),
			outputImage: config.NewImage("im", ""),
			buildConfig: &config.Build{GCSBucket: "bucket", GCSDir: "dir"},
			want:        []string{"-var:output_image_name", "im"},
		},
		{
			testName:    "OutputImageProject",
			inputImage:  config.NewImage("", ""),
			outputImage: config.NewImage("", "proj"),
			buildConfig: &config.Build{GCSBucket: "bucket", GCSDir: "dir"},
			want:        []string{"-var:output_image_project", "proj"},
		},
		{
			testName:    "OutputImageFamily",
			inputImage:  config.NewImage("", ""),
			outputImage: &config.Image{Image: &compute.Image{Family: "family"}, Project: ""},
			buildConfig: &config.Build{GCSBucket: "bucket", GCSDir: "dir"},
			want:        []string{"-var:output_image_family", "family"},
		},
		{
			testName:    "CIData",
			inputImage:  config.NewImage("", ""),
			outputImage: config.NewImage("", ""),
			buildConfig: &config.Build{GCSBucket: "bucket", GCSDir: "dir"},
			want:        []string{"-var:cidata_img"},
		},
		{
			testName:    "DiskSize",
			inputImage:  config.NewImage("", ""),
			outputImage: config.NewImage("", ""),
			buildConfig: &config.Build{DiskSize: 50, GCSBucket: "bucket", GCSDir: "dir"},
			want:        []string{"-var:disk_size_gb", "50"},
		},
		{
			testName:    "GCSPath",
			inputImage:  config.NewImage("", ""),
			outputImage: config.NewImage("", ""),
			buildConfig: &config.Build{GCSBucket: "bucket", GCSDir: "dir"},
			want:        []string{"-gcs_path", "gs://bucket/dir/cos-customizer"},
		},
		{
			testName:    "Project",
			inputImage:  config.NewImage("", ""),
			outputImage: config.NewImage("", ""),
			buildConfig: &config.Build{Project: "proj", GCSBucket: "bucket", GCSDir: "dir"},
			want:        []string{"-project", "proj"},
		},
		{
			testName:    "Zone",
			inputImage:  config.NewImage("", ""),
			outputImage: config.NewImage("", ""),
			buildConfig: &config.Build{Zone: "zone", GCSBucket: "bucket", GCSDir: "dir"},
			want:        []string{"-zone", "zone"},
		},
		{
			testName:    "Timeout",
			inputImage:  config.NewImage("", ""),
			outputImage: config.NewImage("", ""),
			buildConfig: &config.Build{Timeout: "60m", GCSBucket: "bucket", GCSDir: "dir"},
			want:        []string{"-default_timeout", "60m"},
		},
		{
			testName:    "ProvisionerConfigBuildContexts",
			inputImage:  config.NewImage("", ""),
			outputImage: config.NewImage("", ""),
			buildConfig: &config.Build{GCSBucket: "bucket", GCSDir: "dir"},
			provConfig:  &provisioner.Config{},
			wantBuildContexts: map[string]string{
				"user": fmt.Sprintf("gs://bucket/dir/cos-customizer/%s", filepath.Base(files.UserBuildContextArchive)),
			},
		},
		{
			testName:    "ProvisionerConfigSteps",
			inputImage:  config.NewImage("", ""),
			outputImage: config.NewImage("", ""),
			buildConfig: &config.Build{GCSBucket: "bucket", GCSDir: "dir"},
			provConfig: &provisioner.Config{
				Steps: []provisioner.StepConfig{
					{
						Type: "InstallGPU",
						Args: mustMarshalJSON(t, &provisioner.InstallGPUStep{
							GCSDepsPrefix: "gcs_deps",
						}),
					},
				},
			},
			wantSteps: []provisioner.StepConfig{
				{
					Type: "InstallGPU",
					Args: mustMarshalJSON(t, &provisioner.InstallGPUStep{
						GCSDepsPrefix: "gs://bucket/dir/cos-customizer/gcs_files",
					}),
				},
			},
		},
		{
			testName:    "ProvisionerConfigBootDiskReclaimSDA3",
			inputImage:  config.NewImage("", ""),
			outputImage: config.NewImage("", ""),
			buildConfig: &config.Build{GCSBucket: "bucket", GCSDir: "dir", DiskSize: 20},
			provConfig: &provisioner.Config{
				BootDisk: provisioner.BootDiskConfig{
					ReclaimSDA3: true,
				},
			},
			wantBootDisk: &provisioner.BootDiskConfig{
				ReclaimSDA3:       true,
				WaitForDiskResize: true,
			},
		},
		{
			testName:    "ProvisionerConfigBootDiskOEMSize",
			inputImage:  config.NewImage("", ""),
			outputImage: config.NewImage("", ""),
			buildConfig: &config.Build{GCSBucket: "bucket", GCSDir: "dir", DiskSize: 20},
			provConfig: &provisioner.Config{
				BootDisk: provisioner.BootDiskConfig{
					OEMSize: "5G",
				},
			},
			wantBootDisk: &provisioner.BootDiskConfig{
				OEMSize:           "5G",
				WaitForDiskResize: true,
			},
		},
	}
	gcs := fakes.GCSForTest(t)
	defer gcs.Close()
	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			gcs.Objects = make(map[string][]byte)
			gm := &gcsManager{gcs.Client, input.buildConfig.GCSBucket, input.buildConfig.GCSDir}
			if input.provConfig == nil {
				input.provConfig = &provisioner.Config{}
			}
			funcCall := fmt.Sprintf("daisyArgs(_, _, _, %v, %v, %v, %v)", input.inputImage, input.outputImage, input.buildConfig, input.provConfig)
			got, err := daisyArgs(context.Background(), gm, files, input.inputImage, input.outputImage, input.buildConfig, input.provConfig)
			if err != nil {
				t.Fatalf("daisyArgs: %v", err)
			}
			if !isSubSlice(input.want, got) {
				t.Errorf("%s = %v; want subslice %v)", funcCall, got, input.want)
			}
			var provConfig provisioner.Config
			data, err := ioutil.ReadFile(files.ProvConfig)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(data, &provConfig); err != nil {
				t.Fatal(err)
			}
			if input.wantBuildContexts != nil {
				if diff := cmp.Diff(provConfig.BuildContexts, input.wantBuildContexts); diff != "" {
					t.Errorf("%s: build contexts mismatch: diff (-got, +want): %s", funcCall, diff)
				}
			}
			if input.wantSteps != nil {
				if diff := cmp.Diff(provConfig.Steps, input.wantSteps); diff != "" {
					t.Errorf("%s: steps mismatch: diff (-got, +want): %s", funcCall, diff)
				}
			}
			if input.wantBootDisk != nil {
				if diff := cmp.Diff(&provConfig.BootDisk, input.wantBootDisk); diff != "" {
					t.Errorf("%s: steps mismatch: diff (-got, +want): %s", funcCall, diff)
				}
			}
		})
	}
}
