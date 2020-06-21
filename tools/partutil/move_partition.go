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
	"log"
)

// MovePartition moves a partition to a start sector.
// It takes destination input like 2048 (absolute sector number), +5G or -200M.
func MovePartition(disk string, partNumInt int, dest string) error {
	if len(disk) <= 0 || partNumInt <= 0 {
		return errors.New("Error: empty disk name or partition number")
	}

	cmd := fmt.Sprintf("echo %s | sudo sfdisk --move-data=/dev/null %s -N %d", dest, disk, partNumInt)
	if err := ExecCmdToStdout(cmd); Check(err, cmd) {
		return err
	}
	log.Printf("\nCompleted moving %s%d \n\n", disk, partNumInt)
	return nil
}
