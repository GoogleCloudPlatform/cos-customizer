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
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
)

// ExtendPartition extends a partition to a specific end sector.
func ExtendPartition(disk string, partNumInt int, end uint64) error {
	if len(disk) <= 0 || partNumInt <= 0 || end <= 0 {
		return fmt.Errorf("invalid disk name, partition number or end sector, "+
			"input: disk=%q, partNumInt=%d, end sector=%d. ", disk, partNumInt, end)
	}

	// get partition number string
	partNum, err := PartNumIntToString(disk, partNumInt)
	if err != nil {
		return fmt.Errorf("error in converting partition number, "+
			"input: disk=%q, partNumInt=%d, end sector=%d, "+
			"error msg: (%v)", disk, partNumInt, end, err)
	}

	partName := disk + partNum
	var tableBuffer bytes.Buffer

	// dump partition table.
	table, err := ReadPartitionTable(disk)
	if err != nil {
		return fmt.Errorf("cannot read partition table of %q, "+
			"input: disk=%q, partNumInt=%d, end sector=%d, "+
			"error msg: (%v)", disk, disk, partNumInt, end, err)
	}

	var oldSize uint64 = 0
	var newSize uint64 = 0

	// edit partition table.
	table, err = ParsePartitionTable(table, partName, true, func(p *PartContent) {
		oldSize = p.Size
		newSize = end - p.Start + 1
		p.Size = newSize
	})
	if err != nil {
		return fmt.Errorf("error when editing partition table of %q, "+
			"input: disk=%q, partNumInt=%d, end sector=%d, "+
			"error msg: (%v)", disk, disk, partNumInt, end, err)
	}
	if newSize <= oldSize {
		return fmt.Errorf("new size=%d is not larger than the old size=%d, "+
			"input: disk=%q, partNumInt=%d, end sector=%d, "+
			"error msg: (%v)", newSize, oldSize, disk, partNumInt, end, err)
	}

	tableBuffer.WriteString(table)

	// write partition table back.
	writeTableCmd := exec.Command("sudo", "sfdisk", "--no-reread", disk)
	writeTableCmd.Stdin = &tableBuffer
	writeTableCmd.Stdout = os.Stdout
	if err := writeTableCmd.Run(); err != nil {
		return fmt.Errorf("error in writing partition table back to %q, "+
			"input: disk=%q, partNumInt=%d, end sector=%d, "+
			"error msg: (%v)", disk, disk, partNumInt, end, err)
	}

	log.Printf("\nCompleted extending %s\n\n", partName)

	// check and repair file system in the partition.
	cmd := exec.Command("sudo", "e2fsck", "-fp", partName)
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error in checking file system of %q, "+
			"input: disk=%q, partNumInt=%d, end sector=%d, "+
			"error msg: (%v)", partName, disk, partNumInt, end, err)
	}
	log.Printf("\nCompleted checking file system of %s\n\n", partName)

	// resize file system in the partition.
	cmd = exec.Command("sudo", "resize2fs", partName)
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error in resizing file system of %q, "+
			"input: disk=%q, partNumInt=%d, end sector=%d, "+
			"error msg: (%v)", partName, disk, partNumInt, end, err)
	}

	log.Printf("\nCompleted updating file system of %s\n\n", partName)
	return nil
}
