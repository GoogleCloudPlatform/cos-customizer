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
