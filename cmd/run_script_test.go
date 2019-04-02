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
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cos-customizer/fs"

	"github.com/google/subcommands"
)

func createTempFile(dir string) (string, error) {
	file, err := ioutil.TempFile(dir, "")
	if err != nil {
		return "", err
	}
	if err := file.Close(); err != nil {
		os.Remove(file.Name())
		return "", err
	}
	return file.Name(), nil
}

func setupRunScriptFiles() (string, *fs.Files, error) {
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
	files.StateFile, err = createTempFile(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", nil, err
	}
	files.UserBuildContextArchive, err = createTempFile(tmpDir)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", nil, err
	}
	return tmpDir, files, nil
}

func createNonEmptyUserCtxArchive(files *fs.Files, fileName string) error {
	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)
	newFile, err := os.Create(filepath.Join(tmpDir, fileName))
	if err != nil {
		return err
	}
	if err := newFile.Close(); err != nil {
		return err
	}
	if err := os.Remove(files.UserBuildContextArchive); err != nil {
		return err
	}
	return fs.CreateBuildContextArchive(newFile.Name(), files.UserBuildContextArchive)
}

func executeRunScript(files *fs.Files, flags ...string) (subcommands.ExitStatus, error) {
	fs := &flag.FlagSet{}
	runScript := &RunScript{}
	runScript.SetFlags(fs)
	if err := fs.Parse(flags); err != nil {
		return 0, err
	}
	ret := runScript.Execute(nil, fs, files)
	if ret != subcommands.ExitSuccess {
		return ret, fmt.Errorf("RunScript failed. input: %v", flags)
	}
	return ret, nil
}

func TestRunScript(t *testing.T) {
	var testData = []struct {
		testName   string
		flags      []string
		wantPrefix []byte
		wantFiles  int
	}{
		{
			"NoEnv",
			nil,
			[]byte("user\tscript\t\n"),
			0,
		},
		{
			"Env",
			[]string{"-env=HELLO1=world1,HELLO2=world2"},
			[]byte("user\tscript\tuser_env"),
			1,
		},
	}
	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			tmpDir, files, err := setupRunScriptFiles()
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)
			if err := createNonEmptyUserCtxArchive(files, "script"); err != nil {
				t.Fatal(err)
			}
			if _, err := executeRunScript(files, append(input.flags, "-script=script")...); err != nil {
				t.Fatal(err)
			}
			got, err := ioutil.ReadFile(files.StateFile)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.HasPrefix(string(got), string(input.wantPrefix)) {
				t.Errorf("run-script(%v): state file: got %s, want prefix %s", input.flags, string(got), string(input.wantPrefix))
			}
			outputFiles, err := ioutil.ReadDir(files.PersistBuiltinBuildContext)
			if err != nil {
				t.Fatal(err)
			}
			if got := len(outputFiles); got != input.wantFiles {
				t.Logf("unexpected number of output files; got %d, want %d", got, input.wantFiles)
				t.Logf("output files:")
				for _, file := range outputFiles {
					t.Logf("%s", file.Name())
				}
				t.Fail()
			}
		})
	}
}

func TestRunScriptBadScript(t *testing.T) {
	var testData = []struct {
		testName string
		flags    []string
	}{
		{
			"BadScript",
			[]string{"-script=script"},
		},
		{
			"NoScript",
			nil,
		},
	}
	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			tmpDir, files, err := setupRunScriptFiles()
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)
			if got, _ := executeRunScript(files, input.flags...); got == subcommands.ExitSuccess {
				t.Errorf("run-script(%v); got subcommands.ExitSuccess, want failure", input.flags)
			}
		})
	}
}
