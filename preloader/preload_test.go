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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cos-customizer/config"
	"cos-customizer/fakes"
	"cos-customizer/fs"

	"github.com/google/go-cmp/cmp"
	compute "google.golang.org/api/compute/v1"
	yaml "gopkg.in/yaml.v2"
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
	files.BuiltinBuildContextArchive, err = createTempFile(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", nil, err
	}
	files.StateFile, err = createTempFile(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", nil, err
	}
	files.DaisyWorkflow, err = createTempFile(tmpDir)
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
	return tmpDir, files, nil
}

func TestDaisyArgsGCSUpload(t *testing.T) {
	tmpDir, files, err := setupFiles()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	var testData = []struct {
		testName string
		file     string
		contents []byte
	}{
		{
			testName: "UserBuildContextArchive",
			file:     files.UserBuildContextArchive,
			contents: []byte("abc"),
		},
		{
			testName: "BuiltinBuildContextArchive",
			file:     files.BuiltinBuildContextArchive,
			contents: []byte("def"),
		},
		{
			testName: "StateFile",
			file:     files.StateFile,
			contents: []byte("ghi"),
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
			if _, err := daisyArgs(context.Background(), gm, files, config.NewImage("", ""), config.NewImage("", ""), &config.Build{}); err != nil {
				t.Fatalf("daisyArgs: %v", err)
			}
			object := filepath.Base(input.file)
			got, ok := gcs.Objects[fmt.Sprintf("/bucket/cos-customizer/%s", object)]
			if !ok {
				t.Fatalf("daisyArgs: write /bucket/cos-customizer/%s: not found", object)
			}
			if !cmp.Equal(got, input.contents) {
				t.Errorf("daisyArgs: write /bucket/cos-customizer/%s: got %s, want %s", object, string(got), string(input.contents))
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

func TestDaisyArgsCloudConfig(t *testing.T) {
	var testData = []struct {
		testName       string
		startupScript  []byte
		systemdService []byte
		want           map[string]interface{}
	}{
		{
			testName:       "Simple",
			startupScript:  []byte("#!/bin/bash\n\necho \"hello\"\n"),
			systemdService: []byte("[Unit]\nDescription=customizer service\n"),
			want: map[string]interface{}{
				"write_files": []interface{}{
					map[interface{}]interface{}{
						"path":        "/tmp/startup.sh",
						"permissions": "0644",
						"content":     "#!/bin/bash\n\necho \"hello\"\n",
					},
					map[interface{}]interface{}{
						"path":        "/etc/systemd/system/customizer.service",
						"permissions": "0644",
						"content":     "[Unit]\nDescription=customizer service\n",
					},
				},
				"runcmd": []interface{}{
					"echo \"Starting startup service...\"",
					"systemctl daemon-reload",
					"systemctl --no-block start customizer.service",
				},
			},
		},
		{
			testName: "Empty",
			want: map[string]interface{}{
				"write_files": []interface{}{
					map[interface{}]interface{}{
						"path":        "/tmp/startup.sh",
						"permissions": "0644",
						"content":     "",
					},
					map[interface{}]interface{}{
						"path":        "/etc/systemd/system/customizer.service",
						"permissions": "0644",
						"content":     "",
					},
				},
				"runcmd": []interface{}{
					"echo \"Starting startup service...\"",
					"systemctl daemon-reload",
					"systemctl --no-block start customizer.service",
				},
			},
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
			gm := &gcsManager{gcsClient: gcs.Client, gcsBucket: "bucket"}
			if err := ioutil.WriteFile(files.StartupScript, input.startupScript, 0744); err != nil {
				t.Fatal(err)
			}
			if err := ioutil.WriteFile(files.SystemdService, input.systemdService, 0744); err != nil {
				t.Fatal(err)
			}
			args, err := daisyArgs(context.Background(), gm, files, config.NewImage("", ""), config.NewImage("", ""), &config.Build{})
			if err != nil {
				t.Fatalf("daisyArgs: %v", err)
			}
			cloudConfig, ok := getDaisyVarValue("cloud_config", args)
			if !ok {
				t.Fatalf("daisyArgs: could not find \"cloud_config\" variable in args: %v", args)
			}
			data, err := ioutil.ReadFile(cloudConfig)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(string(data), "#cloud-config") {
				t.Fatalf("daisyArgs: cloud config does not have \"#cloud-config\" prefix: %s", string(data))
			}
			got := make(map[string]interface{})
			if err := yaml.Unmarshal(data, got); err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(got, input.want); diff != "" {
				t.Errorf("daisyArgs(_): cloudConfig mismatch: diff (-got +want)\n%s", diff)
			}
		})
	}
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
			outputImage: &config.Image{&compute.Image{Licenses: []string{"my-license"}}, ""},
			buildConfig: &config.Build{GCSBucket: "bucket"},
			workflow:    []byte("{{.Licenses}}"),
			want:        []byte("[\"my-license\"]"),
		},
		{
			testName:    "TwoLicenses",
			outputImage: &config.Image{&compute.Image{Licenses: []string{"license-1", "license-2"}}, ""},
			buildConfig: &config.Build{GCSBucket: "bucket"},
			workflow:    []byte("{{.Licenses}}"),
			want:        []byte("[\"license-1\",\"license-2\"]"),
		},
		{
			testName:    "EmptyStringLicense",
			outputImage: &config.Image{&compute.Image{Licenses: []string{""}}, ""},
			buildConfig: &config.Build{GCSBucket: "bucket"},
			workflow:    []byte("{{.Licenses}}"),
			want:        []byte("null"),
		},
		{
			testName:    "OneEmptyLicense",
			outputImage: &config.Image{&compute.Image{Licenses: []string{"license-1", ""}}, ""},
			buildConfig: &config.Build{GCSBucket: "bucket"},
			workflow:    []byte("{{.Licenses}}"),
			want:        []byte("[\"license-1\"]"),
		},
		{
			testName:    "URLLicense",
			outputImage: &config.Image{&compute.Image{Licenses: []string{"https://www.googleapis.com/compute/v1/projects/my-proj/global/licenses/my-license"}}, ""},
			buildConfig: &config.Build{GCSBucket: "bucket"},
			workflow:    []byte("{{.Licenses}}"),
			want:        []byte("[\"projects/my-proj/global/licenses/my-license\"]"),
		},
		{
			testName:    "Labels",
			outputImage: &config.Image{&compute.Image{Labels: map[string]string{"key": "value"}}, ""},
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
			args, err := daisyArgs(context.Background(), gm, files, config.NewImage("", ""), input.outputImage, input.buildConfig)
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

func TestDaisyArgs(t *testing.T) {
	tmpDir, files, err := setupFiles()
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	var testData = []struct {
		testName    string
		inputImage  *config.Image
		outputImage *config.Image
		buildConfig *config.Build
		want        []string
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
			outputImage: &config.Image{&compute.Image{Family: "family"}, ""},
			buildConfig: &config.Build{GCSBucket: "bucket", GCSDir: "dir"},
			want:        []string{"-var:output_image_family", "family"},
		},
		{
			testName:    "UserBuildContext",
			inputImage:  config.NewImage("", ""),
			outputImage: config.NewImage("", ""),
			buildConfig: &config.Build{GCSBucket: "bucket", GCSDir: "dir"},
			want: []string{
				"-var:user_build_context",
				fmt.Sprintf("gs://bucket/dir/cos-customizer/%s", filepath.Base(files.UserBuildContextArchive)),
			},
		},
		{
			testName:    "BuiltinBuildContext",
			inputImage:  config.NewImage("", ""),
			outputImage: config.NewImage("", ""),
			buildConfig: &config.Build{GCSBucket: "bucket", GCSDir: "dir"},
			want: []string{
				"-var:builtin_build_context",
				fmt.Sprintf("gs://bucket/dir/cos-customizer/%s", filepath.Base(files.BuiltinBuildContextArchive)),
			},
		},
		{
			testName:    "StateFile",
			inputImage:  config.NewImage("", ""),
			outputImage: config.NewImage("", ""),
			buildConfig: &config.Build{GCSBucket: "bucket", GCSDir: "dir"},
			want: []string{
				"-var:state_file",
				fmt.Sprintf("gs://bucket/dir/cos-customizer/%s", filepath.Base(files.StateFile)),
			},
		},
		{
			testName:    "CloudConfig",
			inputImage:  config.NewImage("", ""),
			outputImage: config.NewImage("", ""),
			buildConfig: &config.Build{GCSBucket: "bucket", GCSDir: "dir"},
			want:        []string{"-var:cloud_config"},
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
	}
	gcs := fakes.GCSForTest(t)
	defer gcs.Close()
	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			gcs.Objects = make(map[string][]byte)
			gm := &gcsManager{gcs.Client, input.buildConfig.GCSBucket, input.buildConfig.GCSDir}
			got, err := daisyArgs(context.Background(), gm, files, input.inputImage, input.outputImage, input.buildConfig)
			if err != nil {
				t.Fatalf("daisyArgs: %v", err)
			}
			if !isSubSlice(input.want, got) {
				t.Errorf("daisyArgs(_, _, _, %v, %v, %v) = %v; want subslice %v)", input.inputImage, input.outputImage, input.buildConfig,
					got, input.want)
			}
		})
	}
}
