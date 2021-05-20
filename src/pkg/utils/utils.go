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

// Package utils exports generally useful functionality.
package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
)

// CheckClose closes an io.Closer and checks its error. Useful for checking the
// errors on deferred Close() behaviors.
func CheckClose(closer io.Closer, errMsgOnClose string, err *error) {
	if closeErr := closer.Close(); closeErr != nil {
		var fullErr error
		if errMsgOnClose != "" {
			fullErr = fmt.Errorf("%s: %v", errMsgOnClose, closeErr)
		} else {
			fullErr = closeErr
		}
		if *err == nil {
			*err = fullErr
		} else {
			log.Println(fullErr)
		}
	}
}

// RunCommand runs a command using exec.Command. The command runs in the working
// directory "dir" with environment "env" and outputs to stdout and stderr.
func RunCommand(args []string, dir string, env []string) error {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = dir
	cmd.Env = env
	if err := cmd.Run(); err != nil {
		return fmt.Errorf(`error in cmd "%v", see stderr for details: %v`, args, err)
	}
	return nil
}

// QuoteForShell quotes a string for use in a bash shell.
func QuoteForShell(str string) string {
	return fmt.Sprintf("'%s'", strings.Replace(str, "'", "'\"'\"'", -1))
}

// StringSliceContains returns "true" if elem is in arr, "false" otherwise.
func StringSliceContains(arr []string, elem string) bool {
	for _, s := range arr {
		if s == elem {
			return true
		}
	}
	return false
}
