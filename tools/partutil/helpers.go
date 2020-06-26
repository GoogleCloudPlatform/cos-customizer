// Copyright 2020 Google LLC
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

package partutil

import (
	"errors"
	"log"
	"os"
	"os/exec"
	"strconv"
)

// Check checks whether err is returned and print error message.
func Check(e error, msg string) bool {
	if e != nil {
		log.Printf("ERROR!!!!\n %s\n %s\n\n", msg, e.Error())
		return true
	}
	return false
}

// ExecCmdToStdout runs a command and output to stdout.
func ExecCmdToStdout(cmdLine string) error {
	cmd := exec.Command("/bin/bash", "-c", cmdLine)
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

// ConvertSizeToBytes converts a size string to int unit: bytes.
// It takes a string of number with no unit (sectors), unit B, unit K, unit M, or unit G.
func ConvertSizeToBytes(size string) (int, error) {
	const B = 1
	const K = 1024
	const M = K * 1024
	const G = M * 1024
	const SEC = 512

	var err error
	res := -1
	l := len(size)

	if l <= 0 {
		return -1, errors.New("invalid oemSize: empty string")
	}

	if size[0] < '0' || size[0] > '9' {
		return -1, errors.New("invalid oemSize")
	}

	switch size[l-1] {
	case 'B':
		res, err = strconv.Atoi(size[0 : l-1])
		if Check(err, "cannot convert oemSize to int") {
			return -1, err
		}
		res *= B
	case 'K':
		res, err = strconv.Atoi(size[0 : l-1])
		if Check(err, "cannot convert oemSize to int") {
			return -1, err
		}
		res *= K
	case 'M':
		res, err = strconv.Atoi(size[0 : l-1])
		if Check(err, "cannot convert oemSize to int") {
			return -1, err
		}
		res *= M
	case 'G':
		res, err = strconv.Atoi(size[0 : l-1])
		if Check(err, "cannot convert oemSize to int") {
			return -1, err
		}
		res *= G
	default:
		if size[l-1] >= '0' && size[l-1] <= '9' {
			res, err = strconv.Atoi(size)
			if Check(err, "cannot convert oemSize to int") {
				return -1, err
			}
			res *= SEC
		} else {
			return -1, errors.New("wrong format for oemSize")
		}
	}
	return res, nil
}

// PartNumIntToString converts input int partNumInt into string,
// if disk ends with number, add 'p' to the front.
// Example: /dev/loop5p1
func PartNumIntToString(disk string, partNumInt int) string {
	partNum := strconv.Itoa(partNumInt)
	if disk[len(disk)-1] >= '0' && disk[len(disk)-1] <= '9' {
		partNum = "p" + partNum
	}
	return partNum
}
