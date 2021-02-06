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
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// BuildContext represents the different types of build contexts that are understood
// by the system.
type BuildContext string

const (
	// User represents the user build context.
	User BuildContext = "user"
	// Builtin represents the builtin build context.
	Builtin BuildContext = "builtin"
)

type stateFileEntry struct {
	buildContext BuildContext
	script       string
	env          string
}

func parseStateFileEntry(data string) (*stateFileEntry, error) {
	split := strings.Split(data, "\t")
	if len(split) != 3 {
		return nil, fmt.Errorf("did not find 3 elements in state file entry")
	}
	if split[0] != string(User) && split[0] != string(Builtin) {
		return nil, fmt.Errorf("first field must be a valid build context")
	}
	return &stateFileEntry{BuildContext(split[0]), split[1], split[2]}, nil
}

func (s *stateFileEntry) format() string {
	return fmt.Sprintf("%s\t%s\t%s\n", s.buildContext, s.script, s.env)
}

// CreateStateFile creates the state file.
func CreateStateFile(files *Files) error {
	err := os.MkdirAll(filepath.Dir(files.StateFile), 0774)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(files.StateFile, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return err
	}
	return file.Close()
}

// AppendStateFile appends an entry to the state file.
// The state file encodes a sequence of scripts to run on the preload instance.
func AppendStateFile(stateFile string, buildContext BuildContext, script string, env string) error {
	record := fmt.Sprint((&stateFileEntry{buildContext, script, env}).format())
	writer, err := os.OpenFile(stateFile, os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}
	defer writer.Close()
	_, err = io.WriteString(writer, record)
	return err
}

// StateFileContains checks if an entry exists in the state file with the given build context
// and the given script name.
func StateFileContains(stateFile string, buildContext BuildContext, script string) (bool, error) {
	f, err := os.Open(stateFile)
	if err != nil {
		return false, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		entry, err := parseStateFileEntry(scanner.Text())
		if err != nil {
			return false, err
		}
		if entry.buildContext == buildContext && entry.script == script {
			return true, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return false, err
	}
	return false, nil
}
