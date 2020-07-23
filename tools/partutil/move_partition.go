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
)

// MovePartition moves a partition to a start sector.
// It takes destination input like 2048 (absolute sector number), +5G or -200M.
func MovePartition(disk string, partNumInt int, dest string) error {
	if len(disk) <= 0 || partNumInt <= 0 || len(dest) <= 0 {
		return fmt.Errorf("invalid input: disk=%q, partNumInt=%d, dest=%q", disk, partNumInt, dest)
	}

	var destBuffer bytes.Buffer
	destBuffer.WriteString(dest)
	cmd := exec.Command("sudo", "sfdisk", "--no-reread", "--move-data=/dev/null", disk, "-N", strconv.Itoa(partNumInt))
	cmd.Stdin = &destBuffer
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error in executing sfdisk --move-data, "+
			"input: disk=%q, partNumInt=%d, dest=%q, "+
			"error msg: (%v)", disk, partNumInt, dest, err)
	}
	log.Printf("\nCompleted moving %s partition %d \n\n", disk, partNumInt)
	return nil
}
