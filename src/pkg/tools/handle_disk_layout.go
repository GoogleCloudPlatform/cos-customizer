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
	"fmt"
	"log"
	"strconv"

	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/tools/partutil"
)

// HandleDiskLayout changes the partitions on a COS disk.
// If the auto-update is disabled, it will shrink sda3 to reclaim the space.
// It also moves stateful partition and the OEM partition (if extended) by
// a distance relative to a start point.
//
// If sda3 is shrinked, the start point will be the start of sda3 + sda3Margin (2MB).
// Otherwise, start point will be the original start of the stateful partition.
//
// The stateful partition will be moved to leave enough space for the OEM partition,
// and the OEM partition will be moved to the start point.
// Finally OEM partition will be resized to 1 sector before the new stateful partition.
//
// OEMSize can be the number of sectors (without unit) or size like "3G", "100M", "10000K" or "99999B".
// If there's no need to extend the OEM partition, `oemSize` in the input will be "", a valid input.
func HandleDiskLayout(disk string, statePartNum, oemPartNum int, oemSize string, reclaimSDA3 bool) error {
	if len(disk) <= 0 || statePartNum <= 0 || oemPartNum <= 0 {
		return fmt.Errorf("empty or non-positive input: disk=%q, statePartNum=%d, oemPartNum=%d, oemSize=%q",
			disk, statePartNum, oemPartNum, oemSize)
	}

	if len(oemSize) == 0 {
		oemSize = "0"
	}

	// print the old partition table.
	table, err := partutil.ReadPartitionTable(disk)
	if err != nil {
		return fmt.Errorf("cannot read old partition table of %q, "+
			"input: disk=%q, statePartNum=%d, oemPartNum=%d, oemSize=%q, reclaimSDA3=%t, "+
			"error msg: (%v)", disk, disk, statePartNum, oemPartNum, oemSize, reclaimSDA3, err)
	}
	log.Printf("\nOld partition table:\n%s\n", table)

	// read new size of OEM partition.
	newOEMSizeBytes, err := partutil.ConvertSizeToBytes(oemSize)
	if err != nil {
		return fmt.Errorf("error in reading new OEM size, "+
			"input: disk=%q, statePartNum=%d, oemPartNum=%d, oemSize=%q, reclaimSDA3=%t, "+
			"error msg: (%v)", disk, statePartNum, oemPartNum, oemSize, reclaimSDA3, err)
	}

	// read original size of OEM partition.
	oldOEMSizeSector, err := partutil.ReadPartitionSize(disk, oemPartNum)
	if err != nil {
		return fmt.Errorf("error in reading old OEM size, "+
			"input: disk=%q, statePartNum=%d, oemPartNum=%d, oemSize=%q, reclaimSDA3=%t, "+
			"error msg: (%v)", disk, statePartNum, oemPartNum, oemSize, reclaimSDA3, err)
	}

	oldOEMSizeBytes := oldOEMSizeSector << 9 // change unit to bytes.
	startPointSector, returnAndReboot, err := checkAndReclaimSDA3(disk, statePartNum, reclaimSDA3)
	if err != nil {
		return fmt.Errorf("error in reclaiming sda3, "+
			"input: disk=%q, statePartNum=%d, oemPartNum=%d, oemSize=%q, reclaimSDA3=%t, "+
			"error msg: (%v)", disk, statePartNum, oemPartNum, oemSize, reclaimSDA3, err)
	}
	// need to reboot to reread disk partition table.
	if returnAndReboot {
		return nil
	}

	oemSmaller, err := checkNewOEMSizeSmaller(disk, statePartNum, reclaimSDA3, oldOEMSizeBytes,
		newOEMSizeBytes, startPointSector)
	if err != nil {
		return fmt.Errorf("error in dealing with smaller OEM size, "+
			"input: disk=%q, statePartNum=%d, oemPartNum=%d, oemSize=%q, reclaimSDA3=%t, "+
			"error msg: (%v)", disk, statePartNum, oemPartNum, oemSize, reclaimSDA3, err)
	}
	// No need to resize the OEM partition, everything else done.
	if oemSmaller {
		return nil
	}

	// leave enough space before the stateful partition for the OEM partition.
	// and shrink the OEM partition to make the new start 4k aligned.
	newStateStartSector := partutil.FindLast4KSector(startPointSector + (newOEMSizeBytes >> 9))

	// Move the stateful partition.
	if err := partutil.MovePartition(disk, statePartNum, strconv.FormatUint(newStateStartSector, 10)); err != nil {
		return fmt.Errorf("error in moving stateful partition, "+
			"input: disk=%q, statePartNum=%d, oemPartNum=%d, oemSize=%q, reclaimSDA3=%t, "+
			"error msg: (%v)", disk, statePartNum, oemPartNum, oemSize, reclaimSDA3, err)
	}

	// move OEM partition to the start point.
	if err := partutil.MovePartition(disk, oemPartNum, strconv.FormatUint(startPointSector, 10)); err != nil {
		return fmt.Errorf("error in moving OEM partition, "+
			"input: disk=%q, statePartNum=%d, oemPartNum=%d, oemSize=%q, reclaimSDA3=%t, "+
			"error msg: (%v)", disk, statePartNum, oemPartNum, oemSize, reclaimSDA3, err)
	}
	log.Println("Reclaimed /dev/sda3.")

	// extend the OEM partition.
	if err = partutil.ExtendPartition(disk, oemPartNum, newStateStartSector-1); err != nil {
		return fmt.Errorf("error in extending OEM partition, "+
			"input: disk=%q, statePartNum=%d, oemPartNum=%d, oemSize=%q, reclaimSDA3=%t, "+
			"error msg: (%v)", disk, statePartNum, oemPartNum, oemSize, reclaimSDA3, err)
	}

	// print the new partition table.
	table, err = partutil.ReadPartitionTable(disk)
	if err != nil {
		return fmt.Errorf("cannot read new partition table of %q, "+
			"input: disk=%q, statePartNum=%d, oemPartNum=%d, oemSize=%q, reclaimSDA3=%t, "+
			"error msg: (%v)", disk, disk, statePartNum, oemPartNum, oemSize, reclaimSDA3, err)
	}
	log.Printf("\nCompleted extending OEM partition\n\n New partition table:\n%s\n", table)
	return nil
}

// checkAndReclaimSDA3 checks whether need to reclaim sda3
// and whether it is done.
// If sda3 is shrinked, the start point will be the start of sda3 + sda3Margin (2MB).
// Otherwise, start point will be the original start of the stateful partition.
// It will return startPointSector, returnAndReboot, error.
func checkAndReclaimSDA3(disk string, statePartNum int, reclaimSDA3 bool) (uint64, bool, error) {
	// In some situations, `sfdisk --move-data` requires 1MB free space
	// in the moving direction. Therefore, leaving 2MB after the start of
	// sda3 is a safe choice.
	// Also, this will make sure the start point of the next partition 4K aligned.
	const sda3Margin = 4096 // 2MB margin
	var startPointSector uint64
	var err error
	if reclaimSDA3 {
		// check whether sda3 has already been shrinked.
		minimal, err := partutil.IsPartitionMinimal("/dev/sda", 3)
		if err != nil {
			return 0, false, err
		}
		// not shrinked yet.
		if !minimal {
			_, err = partutil.MinimizePartition("/dev/sda", 3)
			if err != nil {
				return 0, false, fmt.Errorf("error in shrinking sda3, "+
					"error msg: (%v)", err)
			}
			log.Println("Shrinked /dev/sda3.")
			// need to reboot to reload the partition table.
			return 0, true, nil
		}
		// no need to shrink sda3 again.
		sda3StartSector, err := partutil.ReadPartitionStart("/dev/sda", 3)
		if err != nil {
			return 0, false, fmt.Errorf("error in reading the start of sda3, "+
				"error msg: (%v)", err)
		}
		startPointSector = sda3StartSector + sda3Margin // leave enough margin

	} else {
		// start point is the original start sector of the stateful partition.
		startPointSector, err = partutil.ReadPartitionStart(disk, statePartNum)
		if err != nil {
			return 0, false, fmt.Errorf("error in reading the start of old stateful partition, "+
				"error msg: (%v)", err)
		}
	}
	return startPointSector, false, nil
}

// checkNewOEMSizeSmaller checks whether the new OEM size is smaller than the old one.
// If true, move the stateful partition to reclaim sda3 if needed.
// Otherwise, do nothing.
// It returns oemSmaller,error.
func checkNewOEMSizeSmaller(disk string, statePartNum int, reclaimSDA3 bool, oldOEMSizeBytes,
	newOEMSizeBytes, startPointSector uint64) (bool, error) {
	if newOEMSizeBytes <= oldOEMSizeBytes {
		if newOEMSizeBytes != 0 {
			log.Printf("\n!!!!!!!WARNING!!!!!!!\n"+
				"oemSize: %d bytes is not larger than the original OEM partition size: %d bytes, "+
				"nothing is done for the OEM partition.\n ", newOEMSizeBytes, oldOEMSizeBytes)
		}
		if !reclaimSDA3 {
			return true, nil
		}
		// move the stateful partition to the start point.
		if err := partutil.MovePartition(disk, statePartNum, strconv.FormatUint(startPointSector, 10)); err != nil {
			return true, fmt.Errorf("error in moving stateful partition, "+
				"input: disk=%q, statePartNum=%d,reclaimSDA3=%t,oldOEMSizeBytes=%d,newOEMSizeBytes=%d, "+
				"startPointSector=%d, error msg: (%v)", disk, statePartNum, reclaimSDA3,
				oldOEMSizeBytes, newOEMSizeBytes, startPointSector, err)
		}
		log.Println("Reclaimed /dev/sda3.")
		// print the new partition table.
		table, err := partutil.ReadPartitionTable(disk)
		if err != nil {
			return true, fmt.Errorf("cannot read new partition table of %q, "+
				"input: disk=%q, statePartNum=%d,reclaimSDA3=%t,oldOEMSizeBytes=%d,newOEMSizeBytes=%d, "+
				"startPointSector=%d, error msg: (%v)", disk, disk, statePartNum, reclaimSDA3,
				oldOEMSizeBytes, newOEMSizeBytes, startPointSector, err)
		}
		log.Printf("New partition table:\n%s\n", table)
		return true, nil
	}
	return false, nil
}
