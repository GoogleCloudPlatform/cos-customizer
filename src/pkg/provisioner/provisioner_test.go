// Copyright 2021 Google LLC
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

package provisioner

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/fakes"
	"golang.org/x/sys/unix"
)

func testDataDir(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs("testdata")
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func stubMount() {
	mountFunc = func(a1, a2, a3 string, a4 uintptr, a5 string) error {
		return nil
	}
	unmountFunc = func(a1 string, a2 int) error {
		return nil
	}
}

func restoreMount() {
	mountFunc = unix.Mount
	unmountFunc = unix.Unmount
}

func stubMountInfo(filePath, mountPoint string) error {
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		return err
	}
	return ioutil.WriteFile(filePath, []byte(fmt.Sprintf("0 0 0 / %s ro\n", mountPoint)), 0644)
}

func TestStateExists(t *testing.T) {
	ctx := context.Background()
	dir, err := ioutil.TempDir("", "provisioner-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	if err := ioutil.WriteFile(filepath.Join(dir, "state.json"), []byte("{}"), 0660); err != nil {
		t.Fatal(err)
	}
	deps := Deps{
		GCSClient:    nil,
		TarCmd:       "",
		SystemctlCmd: "",
	}
	config := Config{}
	if err := Run(ctx, deps, dir, config); err != errStateAlreadyExists {
		t.Fatalf("Run(ctx, %+v, %q, %+v) = %v; want %v", deps, dir, config, err, errStateAlreadyExists)
	}
}

func TestRunInvalidArgs(t *testing.T) {
	stubMount()
	t.Cleanup(restoreMount)
	tests := []struct {
		name   string
		config Config
	}{
		{
			name: "RunScript",
			config: Config{
				Steps: []StepConfig{
					{
						Type: "RunScript",
						Args: []byte("{}"),
					},
				},
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			tempDir, err := ioutil.TempDir("", "provisioner-test-")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tempDir)
			gcs := fakes.GCSForTest(t)
			deps := Deps{
				GCSClient:    gcs.Client,
				TarCmd:       "tar",
				SystemctlCmd: "/bin/true",
				RootDir:      tempDir,
			}
			stateDir := filepath.Join(tempDir, "var", "lib", ".cos-customizer")
			if err := stubMountInfo(filepath.Join(tempDir, "proc", "self", "mountinfo"), filepath.Join(stateDir, "bin")); err != nil {
				t.Fatal(err)
			}
			funcCall := fmt.Sprintf("Run(ctx, %+v, %q, %+v)", deps, stateDir, test.config)
			if err := Run(ctx, deps, stateDir, test.config); err == nil {
				t.Fatalf("%s = nil; want invalid args", funcCall)
			}
		})
	}
}

func TestRunFailure(t *testing.T) {
	stubMount()
	t.Cleanup(restoreMount)
	testData := testDataDir(t)
	buildCtxDir, err := ioutil.TempDir("", "provisioner-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(buildCtxDir) })
	buildCtx := filepath.Join(buildCtxDir, "test.tar")
	if err := exec.Command("tar", "cf", buildCtx, "-C", filepath.Join(testData, "test_ctx"), ".").Run(); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		gcsObjects map[string]string
		config     Config
	}{
		{
			name: "RunScript",
			gcsObjects: map[string]string{
				"/test/test.tar": buildCtx,
			},
			config: Config{
				BuildContexts: map[string]string{
					"bc": "gs://test/test.tar",
				},
				Steps: []StepConfig{
					{
						Type: "RunScript",
						Args: []byte(`{"BuildContext": "bc", "Path": "run_env.sh"}`),
					},
				},
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			tempDir, err := ioutil.TempDir("", "provisioner-test-")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tempDir)
			gcs := fakes.GCSForTest(t)
			for name, path := range test.gcsObjects {
				data, err := ioutil.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				gcs.Objects[name] = data
			}
			deps := Deps{
				GCSClient:    gcs.Client,
				TarCmd:       "tar",
				SystemctlCmd: "/bin/true",
				RootDir:      tempDir,
			}
			stateDir := filepath.Join(tempDir, "var", "lib", ".cos-customizer")
			if err := stubMountInfo(filepath.Join(tempDir, "proc", "self", "mountinfo"), filepath.Join(stateDir, "bin")); err != nil {
				t.Fatal(err)
			}
			funcCall := fmt.Sprintf("Run(ctx, %+v, %q, %+v)", deps, stateDir, test.config)
			if err := Run(ctx, deps, stateDir, test.config); err == nil {
				t.Fatalf("%s = nil; want err", funcCall)
			}
		})
	}
}

func TestRunSuccess(t *testing.T) {
	stubMount()
	t.Cleanup(restoreMount)
	testData := testDataDir(t)
	buildCtxDir, err := ioutil.TempDir("", "provisioner-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(buildCtxDir) })
	buildCtx := filepath.Join(buildCtxDir, "test.tar")
	if err := exec.Command("tar", "cf", buildCtx, "-C", filepath.Join(testData, "test_ctx"), ".").Run(); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name       string
		gcsObjects map[string]string
		config     Config
	}{
		{
			name:   "EmptyConfig",
			config: Config{},
		},
		{
			name: "RunScript",
			gcsObjects: map[string]string{
				"/test/test.tar": buildCtx,
			},
			config: Config{
				BuildContexts: map[string]string{
					"bc": "gs://test/test.tar",
				},
				Steps: []StepConfig{
					{
						Type: "RunScript",
						Args: []byte(`{"BuildContext": "bc", "Path": "run.sh"}`),
					},
					{
						Type: "RunScript",
						Args: []byte(`{"BuildContext": "bc", "Path": "run_env.sh", "Env": "TEST=t"}`),
					},
				},
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()
			tempDir, err := ioutil.TempDir("", "provisioner-test-")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tempDir)
			gcs := fakes.GCSForTest(t)
			for name, path := range test.gcsObjects {
				data, err := ioutil.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				gcs.Objects[name] = data
			}
			deps := Deps{
				GCSClient:    gcs.Client,
				TarCmd:       "tar",
				SystemctlCmd: "/bin/true",
				RootDir:      tempDir,
			}
			stateDir := filepath.Join(tempDir, "var", "lib", ".cos-customizer")
			if err := stubMountInfo(filepath.Join(tempDir, "proc", "self", "mountinfo"), filepath.Join(stateDir, "bin")); err != nil {
				t.Fatal(err)
			}
			funcCall := fmt.Sprintf("Run(ctx, %+v, %q, %+v)", deps, stateDir, test.config)
			if err := Run(ctx, deps, stateDir, test.config); err != nil {
				t.Fatalf("%s = %v; want nil", funcCall, err)
			}
		})
	}
}
