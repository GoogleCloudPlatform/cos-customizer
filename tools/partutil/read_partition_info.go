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
	"os/exec"
	"strconv"
	"strings"
)

// When we read disk information by dumping the partition table, we get output like the following:
// sudo sfdisk --dump /dev/sdb
// label: gpt
// label-id: 8071096F-DA33-154D-A687-AE097B8252C5
// device: /dev/sdb
// unit: sectors
// first-lba: 2048
// last-lba: 20971486

// /dev/sdb1 : start=     4401152, size=     2097152, type=0FC63DAF-8483-4772-8E79-3D69D8477DE4, uuid=3B41256B-E064-544A-9101-D2647C0B3A38
// /dev/sdb2 : start=      206848, size=     4194304, type=0FC63DAF-8483-4772-8E79-3D69D8477DE4, uuid=60E55EA1-4EEA-9F44-A066-4720F0129089
// /dev/sdb3 : start=     6498304, size=      204800, type=0FC63DAF-8483-4772-8E79-3D69D8477DE4, uuid=9479C34A-49A6-9442-A56F-956396DFAC20

// ReadPartitionSize reads the size of a partition (unit:sectors of 512 Bytes).
func ReadPartitionSize(disk string, partNumInt int) (int, error) {

	if len(disk) <= 0 || partNumInt <= 0 {
		return 0, errors.New("empty input for disk name or partition number")
	}

	// for cases like loop5p1.
	partNum := strconv.Itoa(partNumInt)
	if disk[len(disk)-1] >= '0' && disk[len(disk)-1] <= '9' {
		partNum = "p" + partNum
	}

	// dump partition table and grep the line.
	partName := disk + partNum
	cmd := fmt.Sprintf("sudo sfdisk --dump %s |grep %s", disk, partName)
	line, err := exec.Command("/bin/bash", "-c", cmd).Output()
	if Check(err, cmd) {
		return -1, err
	}
	if len(line) < 4 { // cannot find a valid info line.
		return -1, fmt.Errorf("cannot find partition %s", partName)
	}
	size := -1
	ls := strings.Split(string(line), " ")
	mode := 0
	for _, word := range ls {
		switch mode {
		case 0: // looking for size.
			if word == "size=" {
				mode = 1
			}
		case 1:
			if len(word) > 1 { // a valid size number has at least 1 digits.
				mode = 2
				size, err = strconv.Atoi(word[:len(word)-1]) // a comma at the end.
				if Check(err, "cannot covert size sector to int") {
					return 0, err
				}
			}
		default:
			return -1, errors.New("Error: error in looking for partition")
		}
		if mode == 2 {
			break
		}
	}
	if size == -1 {
		return -1, errors.New("Error: error in looking for partition")
	}
	return size, nil
}

// ReadPartitionStart reads the start sector of a partition.
func ReadPartitionStart(disk string, partNumInt int) (int, error) {
	if len(disk) <= 0 || partNumInt <= 0 {
		return 0, errors.New("empty input for disk name or partition number")
	}

	// for cases like loop5p1.
	partNum := strconv.Itoa(partNumInt)
	if disk[len(disk)-1] >= '0' && disk[len(disk)-1] <= '9' {
		partNum = "p" + partNum
	}

	// dump partition table and grep the line.
	partName := disk + partNum
	cmd := fmt.Sprintf("sudo sfdisk --dump %s |grep %s", disk, partName)
	line, err := exec.Command("/bin/bash", "-c", cmd).Output()
	if Check(err, cmd) {
		return -1, err
	}
	if len(line) < 4 { // cannot find a valid info line.
		return -1, fmt.Errorf("cannot find partition %s", partName)
	}
	start := -1
	ls := strings.Split(string(line), " ")
	mode := 0
	for _, word := range ls {
		switch mode {
		case 0: // looking for start sector.
			if word == "start=" {
				mode = 1
			}
		case 1:
			if len(word) > 1 { // a valid sector number has at least 1 digit.
				mode = 2
				start, err = strconv.Atoi(word[:len(word)-1]) // a comma at the end.
				if Check(err, "cannot covert start sector to int") {
					return 0, err
				}
			}
		default:
			return -1, errors.New("Error: error in looking for partition")
		}
		if mode == 2 {
			break
		}
	}
	if start == -1 {
		return -1, errors.New("Error: error in looking for partition")
	}
	return start, nil
}

// ReadPartitionTable reads the partition table of a disk.
func ReadPartitionTable(disk string) (string, error) {
	cmd := fmt.Sprintf("sudo sfdisk --dump %s", disk)
	table, err := exec.Command("/bin/bash", "-c", cmd).Output()
	if Check(err, cmd) {
		return "", err
	}
	return string(table), nil
}
