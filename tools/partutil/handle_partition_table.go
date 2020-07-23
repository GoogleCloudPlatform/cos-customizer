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

// PartContent contains the info of a partition
type PartContent struct {
	Start uint64
	Size  uint64
}

// ParsePartitionTable takes a partition table and get the start and size of the target partition.
// If change==true, it will rebuild the partition table with data passed in by p *PartContent
// and return the new table
func ParsePartitionTable(table, partName string, change bool, f func(p *PartContent)) (string, error) {
	foundPartition := false
	lines := strings.Split(table, "\n")
	for idx, line := range lines {
		// a white space is needed to prevent cases like /dev/sda14 matches /dev/sda1
		if !strings.HasPrefix(line, partName+" ") {
			continue
		}
		foundPartition = true
		content := strings.Split(line, ":")
		partInfo := strings.Split(content[1], ",")
		startSec := strings.Split(partInfo[0], "=")
		sizeSec := strings.Split(partInfo[1], "=")
		var p PartContent
		var err error
		p.Start, err = strconv.ParseUint(strings.TrimSpace(startSec[1]), 10, 64)
		if err != nil {
			return "", fmt.Errorf("cannot convert %q to int, "+
				"partition info: %q, error msg: (%v)", strings.TrimSpace(startSec[1]), line, err)
		}
		p.Size, err = strconv.ParseUint(strings.TrimSpace(sizeSec[1]), 10, 64)
		if err != nil {
			return "", fmt.Errorf("cannot convert %q to int, "+
				"partition info: %q, error msg: (%v)", strings.TrimSpace(startSec[1]), line, err)
		}
		f(&p)

		// need to rebuild the partition table.
		if change {
			startSec[1] = strconv.FormatUint(p.Start, 10)
			partInfo[0] = strings.Join(startSec, "=")
			sizeSec[1] = strconv.FormatUint(p.Size, 10)
			partInfo[1] = strings.Join(sizeSec, "=")
			content[1] = strings.Join(partInfo, ",")
			lines[idx] = strings.Join(content, ":")
		}
		break
	}
	if !foundPartition {
		return table, fmt.Errorf("cannot find the target partition %q, "+
			"partition table: %s", partName, table)
	}

	if change {
		table = strings.Join(lines, "\n")
	}
	return table, nil
}

// ReadPartitionTable reads the partition table of a disk.
func ReadPartitionTable(disk string) (string, error) {
	table, err := exec.Command("sudo", "sfdisk", "--dump", disk).Output()
	if err != nil {
		return "", fmt.Errorf("cannot dump partition table of %q, "+
			"error msg: (%v)", disk, err)
	}
	return string(table), nil
}

// ReadPartitionSize reads the size of a partition (unit:sectors of 512 Bytes).
func ReadPartitionSize(disk string, partNumInt int) (uint64, error) {
	if len(disk) <= 0 || partNumInt <= 0 {
		return 0, fmt.Errorf("invalid input: disk=%q, partNumInt=%d", disk, partNumInt)
	}

	// get partition number string
	partNum, err := PartNumIntToString(disk, partNumInt)
	if err != nil {
		return 0, fmt.Errorf("error in converting partition number, "+
			"input: disk=%q, partNumInt=%d, "+
			"error msg: (%v)", disk, partNumInt, err)
	}
	partName := disk + partNum

	table, err := ReadPartitionTable(disk)
	if err != nil {
		return 0, fmt.Errorf("cannot read partition table of %q, "+
			"input: disk=%q, partNumInt=%d, "+
			"error msg: (%v)", disk, disk, partNumInt, err)
	}

	var size uint64 = 0
	if _, err = ParsePartitionTable(table, partName, false, func(p *PartContent) { size = p.Size }); err != nil {
		return 0, fmt.Errorf("error parsing partition table of %q, "+
			"input: disk=%q, partNumInt=%d, "+
			"error msg: (%v)", disk, disk, partNumInt, err)
	}
	return size, nil
}

// ReadPartitionStart reads the start sector of a partition.
func ReadPartitionStart(disk string, partNumInt int) (uint64, error) {
	if len(disk) <= 0 || partNumInt <= 0 {
		return 0, fmt.Errorf("invalid input: disk=%q, partNumInt=%d", disk, partNumInt)
	}

	// get partition number string
	partNum, err := PartNumIntToString(disk, partNumInt)
	if err != nil {
		return 0, fmt.Errorf("error in converting partition number, "+
			"input: disk=%q, partNumInt=%d, "+
			"error msg: (%v)", disk, partNumInt, err)
	}
	partName := disk + partNum

	table, err := ReadPartitionTable(disk)
	if err != nil {
		return 0, fmt.Errorf("cannot read partition table of %q, "+
			"input: disk=%q, partNumInt=%d, "+
			"error msg: (%v)", disk, disk, partNumInt, err)
	}

	var start uint64 = 0
	if _, err = ParsePartitionTable(table, partName, false, func(p *PartContent) { start = p.Start }); err != nil {
		return 0, fmt.Errorf("error parsing partition table of %q, "+
			"input: disk=%q, partNumInt=%d, "+
			"error msg: (%v)", disk, disk, partNumInt, err)
	}
	return start, nil
}
