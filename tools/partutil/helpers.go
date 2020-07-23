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
	"fmt"
	"strconv"
)

// ConvertSizeToBytes converts a size string to int unit: bytes.
// It takes a string of number with no unit (sectors), unit B, unit K, unit M, or unit G.
func ConvertSizeToBytes(size string) (uint64, error) {
	const B = 1
	const K = 1024
	const M = K * 1024
	const G = M * 1024
	const Sec = 512

	var err error
	var res uint64 = 0
	l := len(size)

	if l <= 0 {
		return 0, errors.New("invalid oemSize: empty string")
	}

	if size[0] < '0' || size[0] > '9' {
		return 0, fmt.Errorf("invalid oemSize, the first char should be digit, "+
			"input size: %q", size)
	}

	if size[l-1] >= '0' && size[l-1] <= '9' {
		res, err = strconv.ParseUint(size, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("cannot convert %q to int", size)
		}
		res *= Sec
	} else {
		res, err = strconv.ParseUint(size[0:l-1], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("cannot convert %q in input: %q to int", string(size[0:l-1]), size)
		}
		switch size[l-1] {
		case 'B':
			res *= B
		case 'K':
			res *= K
		case 'M':
			res *= M
		case 'G':
			res *= G
		default:
			return 0, fmt.Errorf("wrong format for oemSize, input: %q, "+
				"expecting input like 10G, 200M, 600K, 5000B or 1024", size)
		}
	}

	return res, nil
}

// ConvertSizeToGBRoundUp converts input size to GB unit.
// Rounded up, since extend disk can only take GB unit.
// Used by Daisy workflow to resize the disk.
func ConvertSizeToGBRoundUp(size string) (uint64, error) {
	sizeByte, err := ConvertSizeToBytes(size)
	if err != nil {
		return 0, err
	}
	sizeGB := sizeByte >> 30
	if (sizeGB << 30) != sizeByte {
		sizeGB++
	}
	return sizeGB, nil
}

// PartNumIntToString converts input int partNumInt into string,
// if disk ends with number, add 'p' to the front.
// Example: /dev/loop5p1
func PartNumIntToString(disk string, partNumInt int) (string, error) {
	if len(disk) <= 0 {
		return "", errors.New("empty disk name")
	}
	partNum := strconv.Itoa(partNumInt)
	if disk[len(disk)-1] >= '0' && disk[len(disk)-1] <= '9' {
		partNum = "p" + partNum
	}
	return partNum, nil
}
