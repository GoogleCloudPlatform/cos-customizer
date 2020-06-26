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
	"strconv"
	"strings"
)

// ExtendPartition extends a partition to a specific end sector.
func ExtendPartition(disk string, partNumInt, end int) error {
	if len(disk) <= 0 || partNumInt <= 0 || end <= 0 {
		return fmt.Errorf("invalid disk name, partition number or end sector, "+
			"input: disk=%s, partNumInt=%d, end sector=%d. ", disk, partNumInt, end)
	}

	// get partition number string
	partNum, err := PartNumIntToString(disk, partNumInt)
	if err != nil {
		return fmt.Errorf("error in converting partition number, "+
			"input: disk=%s, partNumInt=%d, end sector=%d, "+
			"error msg: (%v)", disk, partNumInt, end, err)
	}

	partName := disk + partNum
	var tableBuffer bytes.Buffer

	// dump partition table.
	tableByte, err := exec.Command("sudo", "sfdisk", "--dump", disk).Output()
	if err != nil {
		return fmt.Errorf("error in dumping partition table of %s, "+
			"input: disk=%s, partNumInt=%d, end sector=%d, "+
			"error msg: (%v)", disk, disk, partNumInt, end, err)
	}

	// edit partition table.
	tableString, err := editPartitionEnd(string(tableByte), partName, end)
	if err != nil {
		return fmt.Errorf("error when editing partition table of %s to %d, "+
			"input: disk=%s, partNumInt=%d, end sector=%d, "+
			"error msg: (%v)", disk, end, disk, partNumInt, end, err)
	}

	tableBuffer.WriteString(tableString)

	// write partition table back.
	writeTableCmd := exec.Command("sudo", "sfdisk", "--no-reread", disk)
	writeTableCmd.Stdin = &tableBuffer
	writeTableCmd.Stdout = os.Stdout
	if err := writeTableCmd.Run(); err != nil {
		return fmt.Errorf("error in writing partition table back to %s, "+
			"input: disk=%s, partNumInt=%d, end sector=%d, "+
			"error msg: (%v)", disk, disk, partNumInt, end, err)
	}

	log.Printf("\nCompleted extending %s\n\n", partName)

	// check and repair file system in the partition.
	cmd := fmt.Sprintf("sudo e2fsck -fp %s", partName)
	if err := ExecCmdToStdout(cmd); err != nil {
		return fmt.Errorf("error in checking file system of %s, "+
			"input: disk=%s, partNumInt=%d, end sector=%d, "+
			"error msg: (%v)", partName, disk, partNumInt, end, err)
	}
	log.Printf("\nCompleted checking file system of %s\n\n", partName)

	// resize file system in the partition.
	cmd = fmt.Sprintf("sudo resize2fs %s", partName)
	if err := ExecCmdToStdout(cmd); err != nil {
		return fmt.Errorf("error in resizing file system of %s, "+
			"input: disk=%s, partNumInt=%d, end sector=%d, "+
			"error msg: (%v)", partName, disk, partNumInt, end, err)
	}

	log.Printf("\nCompleted updating file system of %s\n\n", partName)
	return nil
}

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

// editPartitionEnd changes partition end in the partition table string to extend the partition.
func editPartitionEnd(table, partName string, end int) (string, error) {
	var err error
	lines := strings.Split(table, "\n")
	haveValidPartition := false // whether has the required partition.
	for i, line := range lines {
		if strings.Contains(line, partName) {
			ls := strings.Split(line, " ")
			mode := 0
			start := -1
			for j, word := range ls {
				switch mode {
				case 0: // looking for start sector.
					if word == "start=" {
						mode = 1
					}
				case 1:
					if len(word) > 1 { // a valid sector number has at least 1 digits.
						mode = 2
						start, err = strconv.Atoi(word[:len(word)-1]) // a comma at the end.
						if err != nil {
							return "", fmt.Errorf("cannot convert start sector to int, "+
								"start string: %s, "+
								"input: partName=%s, end=%d, "+
								"error msg: (%v)", word, partName, end, err)
						}
					}
				case 2:
					if word == "size=" {
						mode = 3
					}
				case 3:
					if len(word) > 1 { // a valid sector number has at least 1 digits.

						size, err := strconv.Atoi(word[:len(word)-1]) // a comma at the end.
						if err != nil {
							return "", fmt.Errorf("cannot convert start sector to int, "+
								"size string: %s, "+
								"input: partName=%s, end=%d, "+
								"error msg: (%v)", word, partName, end, err)
						}
						if end-start+1 <= size {
							return "", fmt.Errorf("new size is smaller or equal to the original size, "+
								"old size: %d, new size: %d"+
								"input: partName=%s, end=%d, "+
								"error msg: (%v)", size, end-start+1, partName, end, err)
						}
						haveValidPartition = true // Modification completed.
						ls[j] = strconv.Itoa(end+1-start) + ","
					}
				default:
					return "", fmt.Errorf("error in looking for valid info, "+
						"input: partName=%s, end=%d", partName, end)
				}
				if haveValidPartition {
					break
				}
			}

			// recreate the line.
			if haveValidPartition {
				lines[i] = strings.Join(ls, " ")
			}
			break
		}
	}
	if !haveValidPartition {
		return "", fmt.Errorf("partition not founddd, "+
			"input: partName=%s, end=%d", partName, end)
	}
	// recreate the partition table.
	table = strings.Join(lines, "\n")
	return table, nil
}
