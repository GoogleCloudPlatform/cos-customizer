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

package fs

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func diffDirs(got, want string) (string, error) {
	cmd := exec.Command("diff", "-r", "-q", got, want)
	diff, err := cmd.CombinedOutput()
	if err != nil {
		log.Println(string(diff))
		return "", err
	}
	return string(diff), nil
}

func archiveMatchesPath(archive, path string) (string, error) {
	got, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(got)
	cmd := exec.Command("tar", "xvf", archive, "-C", got)
	if err := cmd.Run(); err != nil {
		return "", err
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return diffDirs(got, path)
	}
	if info.Mode().IsRegular() {
		want, err := ioutil.TempDir("", "")
		if err != nil {
			return "", err
		}
		defer os.RemoveAll(want)
		cmd := exec.Command("cp", path, want)
		if err := cmd.Run(); err != nil {
			return "", err
		}
		return diffDirs(got, want)
	}
	return "", fmt.Errorf("path %s is not a directory or regular file", path)
}

func TestCreateBuildContextArchiveEmptyDir(t *testing.T) {
	outputDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(outputDir)
	emptyDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(emptyDir)
	if err := CreateBuildContextArchive(emptyDir, filepath.Join(outputDir, "archive")); err != nil {
		t.Log("CreateBuildContextArchive(emptyDir, _)")
		t.Fatal(err)
	}
	diff, err := archiveMatchesPath(filepath.Join(outputDir, "archive"), emptyDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff) > 0 {
		t.Errorf("CreateBuildContextArchive(emptyDir, _), diff: %s, want: emptyDir", diff)
	}
}

func TestCreateBuildContextArchiveSelfReferential(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	testdata := filepath.Join(tmpDir, "test_1")
	if err := CopyRecursive("testdata/test_1", testdata); err != nil {
		t.Fatal(err)
	}
	if err := CreateBuildContextArchive(testdata, filepath.Join(testdata, "archive")); err != nil {
		t.Logf("CreateBuildContextArchive(%s, _)", testdata)
		t.Fatal(err)
	}
	archive, err := ioutil.ReadFile(filepath.Join(testdata, "archive"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(testdata, "archive")); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(filepath.Join(tmpDir, "archive"), archive, 0664); err != nil {
		t.Fatal(err)
	}
	diff, err := archiveMatchesPath(filepath.Join(tmpDir, "archive"), testdata)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff) > 0 {
		t.Errorf("CreateBuildContextArchive(%s, _), diff: %s, want: %s", testdata, diff, testdata)
	}
}

func TestCreateBuildContextArchive(t *testing.T) {
	testData := []struct {
		testName string
		path     string
	}{
		{"RegularFiles", "testdata/test_1"},
		{"RegFilesAndDirs", "testdata/test_2"},
		{"RegularFile", "testdata/test_3"},
	}
	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			tmpDir, err := ioutil.TempDir("", "")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)
			if err := CreateBuildContextArchive(input.path, filepath.Join(tmpDir, "archive")); err != nil {
				t.Logf("CreateBuildContextArchive(%s, _)", input.path)
				t.Fatal(err)
			}
			diff, err := archiveMatchesPath(filepath.Join(tmpDir, "archive"), input.path)
			if err != nil {
				t.Fatal(err)
			}
			if len(diff) > 0 {
				t.Errorf("CreateBuildContextArchive(%s, _), diff: %s, want: %s", input.path, diff, input.path)
			}
		})
	}
}

func TestArchiveHasObjectEmptyDir(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	if err := CreateBuildContextArchive(tmpDir, filepath.Join(tmpDir, "archive")); err != nil {
		t.Fatal(err)
	}
	actual, err := ArchiveHasObject(filepath.Join(tmpDir, "archive"), "a")
	if err != nil {
		t.Fatal(err)
	}
	if actual != false {
		t.Errorf("ArchiveHasObject(emptyArchive, a) = %t, want: false", actual)
	}
}

func TestArchiveHasObject(t *testing.T) {
	testData := []struct {
		testName string
		path     string
		object   string
		expected bool
	}{
		{"DirWithoutFile", "testdata/test_1", "d", false},
		{"DirWithFile", "testdata/test_1", "c", true},
		{"DirWithDir", "testdata/test_2", "a/", true},
		{"DirWithDirInvalid", "testdata/test_2", "a", false},
		{"DirWithNestedFile", "testdata/test_2", "a/a", true},
		{"RegularFile", "testdata/test_3", "test_3", true},
		{"EmptyQuery", "testdata/test_3", "", false},
	}
	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			tmpDir, err := ioutil.TempDir("", "")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)
			if err := CreateBuildContextArchive(input.path, filepath.Join(tmpDir, "archive")); err != nil {
				t.Fatal(err)
			}
			actual, err := ArchiveHasObject(filepath.Join(tmpDir, "archive"), input.object)
			if err != nil {
				t.Fatal(err)
			}
			if actual != input.expected {
				t.Errorf("ArchiveHasObject(%s, %s) = %t, want: %t", input.path, input.object, actual,
					input.expected)
			}
		})
	}
}
