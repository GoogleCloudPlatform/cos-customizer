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
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// ExtendPartition extends a partition to a specific end sector.
func ExtendPartition(disk string, partNumInt, end int) error {
	if len(disk) <= 0 || partNumInt <= 0 {
		return errors.New("invalid disk name or partition number")
	}

	// for cases like loop5p1.
	partNum := strconv.Itoa(partNumInt)
	if disk[len(disk)-1] >= '0' && disk[len(disk)-1] <= '9' {
		partNum = "p" + partNum
	}

	partName := disk + partNum
	var tableBuffer bytes.Buffer

	// dump partition table.
	cmd := fmt.Sprintf("sudo sfdisk --dump %s", disk)
	tableByte, err := exec.Command(cmd).Output()
	if Check(err, "error in dumping partition table") {
		return err
	}

	// edit partition table.
	tableString, err := editPartitionTableFile(string(tableByte), partName, end)
	if Check(err, fmt.Sprintf("editing partition table file of %s to ending sector at: %d", partName, end)) {
		return err
	}

	tableBuffer.WriteString(tableString)

	// write partition table back.
	cmd = fmt.Sprintf("sudo sfdisk --no-reread %s", disk)
	writeTableCmd := exec.Command(cmd)
	writeTableCmd.Stdin = &tableBuffer
	writeTableCmd.Stdout = os.Stdout
	if err := writeTableCmd.Run(); err != nil {
		log.Printf("error in writing partition table back to %s \n", disk)
	}

	log.Printf("\nCompleted extending %s\n\n", partName)

	// check and repair file system in the partition.
	cmd = fmt.Sprintf("sudo e2fsck -fp %s", partName)
	if err := ExecCmdToStdout(cmd); Check(err, cmd) {
		return err
	}
	log.Printf("\nCompleted checking file system of %s\n\n", partName)

	// resize file system in the partition.
	cmd = fmt.Sprintf("sudo resize2fs %s", partName)
	if err := ExecCmdToStdout(cmd); Check(err, cmd) {
		return err
	}

	log.Printf("\nCompleted updating file system of %s\n\n", partName)
	return nil
}

// change partition table file to extend partition.
func editPartitionTableFile(table, partName string, end int) (string, error) {
	var err error
	lines := strings.Split(table, "\n")
	have := false // whether has valid information about the partition.
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
						if Check(err, "cannot convert start sector to int") {
							return "", err
						}
					}
				case 2:
					if word == "size=" {
						mode = 3
					}
				case 3:
					if len(word) > 1 { // a valid sector number has at least 1 digits.

						size, err := strconv.Atoi(word[:len(word)-1]) // a comma at the end.
						if Check(err, "cannot convert size to int") {
							return "", err
						}
						if end-start+1 <= size {
							return "", errors.New("Error: new size is not larger than the original size")
						}
						have = true // Modification completed.
						ls[j] = strconv.Itoa(end+1-start) + ","
					}
				default:
					return "", errors.New("Error: error in looking for partition")
				}
				if have {
					break
				}
			}

			// recreate the line.
			if have {
				lines[i] = strings.Join(ls, " ")
			}
			break
		}
	}
	if !have {
		return "", errors.New("Error: Partition not found")
	}
	// recreate the partition table.
	table = strings.Join(lines, "\n")
	return table, nil
}
