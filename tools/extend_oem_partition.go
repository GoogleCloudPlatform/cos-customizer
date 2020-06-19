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
	"errors"
	"fmt"
	"log"
	"strconv"
)

// ExtendOemPartition moves stateful partition towards the end of the disk
// Then move oem partition to the original place of the stateful partition
// Finally resize the oem partition to 1 sector before the new stateful partition
// oemSize can be the number of sectors (without unit) or size like "3G", "100M", "10000K" or "99999B"
func ExtendOemPartition(disk string, statePartNum, oemPartNum int, oemSize string) error {
	const SECTOR = 512

	if len(disk) <= 0 || statePartNum <= 0 || oemPartNum <= 0 {
		return errors.New("invalid input for disk name or partition number")
	}

	// read new size of OEM partition.
	newOemSizeBytes, err := partutil.ConvertSizeToBytes(oemSize)

	// read original size of OEM partition.
	oldOemSize, err := partutil.ReadPartitionSize(disk, oemPartNum)
	if err != nil {
		return err
	}
	oldOemSizeBytes := oldOemSize * SECTOR // change unit to bytes.

	if err != nil {
		return err
	}

	if newOemSizeBytes == -1 {
		return fmt.Errorf("Error: invalid oemSize: %s", oemSize)
	}

	if newOemSizeBytes <= oldOemSizeBytes {
		log.Printf("\n!!!!!!!WARNING!!!!!!!:\noemSize: %d bytes is not larger than the original OEM partition size: %d bytes, nothing is done\n", newOemSizeBytes, oldOemSizeBytes)
		return nil
	}

	// print the old partition table.
	table, err := partutil.ReadPartitionTable(disk)
	if err != nil {
		return fmt.Errorf("Error: cannot read partition table of %s", disk)
	}
	log.Printf("\nOld partition table:\n%s\n", table)

	// record the original start sector of the stateful partition.
	oldStartSector, err := partutil.ReadPartitionStart(disk, statePartNum)
	if err != nil {
		return err
	}

	// move the stateful partition.
	if err := partutil.MovePartition(disk, statePartNum, "+"+oemSize); err != nil {
		return err
	}

	// move oem partition to the original start sector of the stateful partition.

	if err := partutil.MovePartition(disk, oemPartNum, strconv.Itoa(oldStartSector)); err != nil {
		return err
	}

	// record the new start sector of the stateful partition.
	newStartSector, err := partutil.ReadPartitionStart(disk, statePartNum)
	if err != nil {
		return err
	}

	// extend the oem partition.
	if err = partutil.ExtendPartition(disk, oemPartNum, newStartSector-1); err != nil {
		return err
	}

	// print the new partition table.
	table, err = partutil.ReadPartitionTable(disk)
	if err != nil {
		return fmt.Errorf("Error: cannot read partition table of %s", disk)
	}
	log.Printf("\nCompleted extending OEM partition\n\n New partition table:\n%s\n", table)
	return nil
}
