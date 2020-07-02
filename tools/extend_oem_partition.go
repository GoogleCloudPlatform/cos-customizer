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

package tools

import (
	"cos-customizer/tools/partutil"
	"fmt"
	"log"
	"strconv"
)

// ExtendOEMPartition moves stateful partition towards the end of the disk
// Then move OEM partition to the original place of the stateful partition
// Finally resize the OEM partition to 1 sector before the new stateful partition
// OEMSize can be the number of sectors (without unit) or size like "3G", "100M", "10000K" or "99999B"
func ExtendOEMPartition(disk string, statePartNum, oemPartNum int, oemSize string) error {
	const SECTOR = 512

	if len(disk) <= 0 || statePartNum <= 0 || oemPartNum <= 0 || len(oemSize) <= 0 {
		return fmt.Errorf("empty or non-positive input: disk=%s, statePartNum=%d, oemPartNum=%d, oemSize=%s",
			disk, statePartNum, oemPartNum, oemSize)
	}

	// read new size of OEM partition.
	newOEMSizeBytes, err := partutil.ConvertSizeToBytes(oemSize)
	if err != nil {
		return fmt.Errorf("error in reading new OEM size, "+
			"input: disk=%s, statePartNum=%d, oemPartNum=%d, oemSize=%s, "+
			"error msg: (%v)", disk, statePartNum, oemPartNum, oemSize, err)
	}

	// read original size of OEM partition.
	oldOEMSize, err := partutil.ReadPartitionSize(disk, oemPartNum)
	if err != nil {
		return fmt.Errorf("error in reading old OEM size, "+
			"input: disk=%s, statePartNum=%d, oemPartNum=%d, oemSize=%s, "+
			"error msg: (%v)", disk, statePartNum, oemPartNum, oemSize, err)
	}
	oldOEMSizeBytes := oldOEMSize * SECTOR // change unit to bytes.

	if newOEMSizeBytes <= oldOEMSizeBytes {
		log.Printf("\n!!!!!!!WARNING!!!!!!!\n"+
			"oemSize: %d bytes is not larger than the original OEM partition size: %d bytes, "+
			"nothing is done\n "+
			"input: disk=%s, statePartNum=%d, oemPartNum=%d, oemSize=%s",
			newOEMSizeBytes, oldOEMSizeBytes, disk, statePartNum, oemPartNum, oemSize)
		return nil
	}

	// print the old partition table.
	table, err := partutil.ReadPartitionTable(disk)
	if err != nil {
		return fmt.Errorf("cannot read old partition table of %s, "+
			"input: disk=%s, statePartNum=%d, oemPartNum=%d, oemSize=%s, "+
			"error msg: (%v)", disk, disk, statePartNum, oemPartNum, oemSize, err)
	}
	log.Printf("\nOld partition table:\n%s\n", table)

	// record the original start sector of the stateful partition.
	oldStateStartSector, err := partutil.ReadPartitionStart(disk, statePartNum)
	if err != nil {
		return fmt.Errorf("cannot read old stateful partition start, "+
			"input: disk=%s, statePartNum=%d, oemPartNum=%d, oemSize=%s, "+
			"error msg: (%v)", disk, statePartNum, oemPartNum, oemSize, err)
	}

	// move the stateful partition.
	if err := partutil.MovePartition(disk, statePartNum, "+"+oemSize); err != nil {
		return fmt.Errorf("error in moving stateful partition, "+
			"input: disk=%s, statePartNum=%d, oemPartNum=%d, oemSize=%s, "+
			"error msg: (%v)", disk, statePartNum, oemPartNum, oemSize, err)
	}

	// record the new start sector of the stateful partition.
	newStateStartSector, err := partutil.ReadPartitionStart(disk, statePartNum)
	if err != nil {
		return fmt.Errorf("cannot read new stateful partition start, "+
			"input: disk=%s, statePartNum=%d, oemPartNum=%d, oemSize=%s, "+
			"error msg: (%v)", disk, statePartNum, oemPartNum, oemSize, err)
	}

	// move OEM partition to the original start sector of the stateful partition.
	if err := partutil.MovePartition(disk, oemPartNum, strconv.Itoa(oldStateStartSector)); err != nil {
		return fmt.Errorf("error in moving OEM partition, "+
			"input: disk=%s, statePartNum=%d, oemPartNum=%d, oemSize=%s, "+
			"error msg: (%v)", disk, statePartNum, oemPartNum, oemSize, err)
	}

	// extend the OEM partition.
	if err = partutil.ExtendPartition(disk, oemPartNum, newStateStartSector-1); err != nil {
		return fmt.Errorf("error in extending OEM partition, "+
			"input: disk=%s, statePartNum=%d, oemPartNum=%d, oemSize=%s, "+
			"error msg: (%v)", disk, statePartNum, oemPartNum, oemSize, err)
	}

	// print the new partition table.
	table, err = partutil.ReadPartitionTable(disk)
	if err != nil {
		return fmt.Errorf("cannot read new partition table of %s, "+
			"input: disk=%s, statePartNum=%d, oemPartNum=%d, oemSize=%s, "+
			"error msg: (%v)", disk, disk, statePartNum, oemPartNum, oemSize, err)
	}
	log.Printf("\nCompleted extending OEM partition\n\n New partition table:\n%s\n", table)
	return nil
}
