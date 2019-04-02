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
	"io/ioutil"
	"os"
	"testing"
)

func TestCreateStateFileDoesntExist(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	files := &Files{StateFile: tmpFile.Name()}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpFile.Name())
		t.Fatal(err)
	}
	if err := os.Remove(tmpFile.Name()); err != nil {
		t.Fatal(err)
	}
	if err := CreateStateFile(files); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(files.StateFile)
	if _, err := os.Stat(files.StateFile); os.IsNotExist(err) {
		t.Errorf("state file was not created. Expected state file to be created.")
	}
}

func TestCreateStateFileAlreadyExists(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()
	files := &Files{StateFile: tmpFile.Name()}
	if err := CreateStateFile(files); err == nil {
		t.Errorf("no error returned when state file already exists.")
	}
}

func TestAppendStateFile(t *testing.T) {
	testAppendStateFileData := []struct {
		testName  string
		context   BuildContext
		script    string
		env       string
		stateFile string
		expected  string
	}{
		{"WithEnv", User, "script", "env", "", "user\tscript\tenv\n"},
		{"NoEnv", Builtin, "script", "", "", "builtin\tscript\t\n"},
		{"NotEmpty", User, "script", "", "builtin\tscript\tenv\n", "builtin\tscript\tenv\nuser\tscript\t\n"},
	}
	for _, input := range testAppendStateFileData {
		t.Run(input.testName, func(t *testing.T) {
			tmpFile, err := ioutil.TempFile("", "")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpFile.Name())
			_, err = tmpFile.WriteString(input.stateFile)
			if err != nil {
				tmpFile.Close()
				t.Fatal(err)
			}
			err = tmpFile.Close()
			if err != nil {
				t.Fatal(err)
			}
			if err := AppendStateFile(tmpFile.Name(), input.context, input.script, input.env); err != nil {
				t.Fatal(err)
			}
			actual, err := ioutil.ReadFile(tmpFile.Name())
			if err != nil {
				t.Fatal(err)
			}
			if string(actual) != input.expected {
				t.Errorf("actual: %s expected: %s", string(actual), input.expected)
			}
		})
	}
}

func TestStateFileContains(t *testing.T) {
	testStateFileContainsData := []struct {
		testName  string
		context   BuildContext
		script    string
		stateFile string
		expected  bool
	}{
		{"EmptyFile", User, "script", "", false},
		{"NonEmptyHasInput", User, "script", "user\tscript\t\n", true},
		{"WrongScript", User, "script", "user\tother\t\n", false},
		{"WrongContext", User, "script", "builtin\tscript\t\n", false},
		{"WrongScriptContext", User, "script", "builtin\tother\t\n", false},
		{"MatchSecondEntry", User, "script", "builtin\tother\t\nuser\tscript\t\n", true},
		{"MatchFirstEntry", User, "script", "user\tscript\t\nbuiltin\tother\t\n", true},
		{"MatchSecondEntryBuiltin", Builtin, "script", "user\tscript\t\nbuiltin\tscript\t\n", true},
	}
	for _, input := range testStateFileContainsData {
		t.Run(input.testName, func(t *testing.T) {
			tmpFile, err := ioutil.TempFile("", "")
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(tmpFile.Name())
			_, err = tmpFile.WriteString(input.stateFile)
			if err != nil {
				tmpFile.Close()
				t.Fatal(err)
			}
			if err := tmpFile.Close(); err != nil {
				t.Fatal(err)
			}
			actual, err := StateFileContains(tmpFile.Name(), input.context, input.script)
			if err != nil {
				t.Fatal(err)
			}
			if actual != input.expected {
				t.Errorf("actual: %v expected: %v", actual, input.expected)
			}
		})
	}
}
