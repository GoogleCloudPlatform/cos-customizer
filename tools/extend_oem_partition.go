package tools

import (
	"cos-customizer/tools/partutil"
	"errors"
	"log"
	"strconv"
)

//ExtendOemPartition moves stateful partition towards the end of the disk
//Then move oem partition to the original place of the stateful partition
//Finally resize the oem partition to 1 sector before the new stateful partition
//oemSize can be the number of sectors (without unit) or size like "3G", "100M", "10000K" or "99999B"
func ExtendOemPartition(disk string, statePartNum, oemPartNum int, oemSize string) error {
	const SECTOR = 512

	if len(disk) <= 0 || statePartNum <= 0 || oemPartNum <= 0 {
		return errors.New("invalid input for disk name or partition number")
	}

	//read new size of OEM partition
	newOemSizeBytes, err := partutil.ConvertSizeToBytes(oemSize)

	//read original size of OEM partition
	oriOemSize, err := partutil.ReadPartitionSize(disk, oemPartNum)
	if err != nil {
		return err
	}
	oriOemSizeBytes := oriOemSize * SECTOR //change unit to bytes

	if err != nil {
		return err
	}

	if newOemSizeBytes <= oriOemSizeBytes {
		return errors.New("Error: oemSize: " + strconv.Itoa(newOemSizeBytes) + " bytes is not larger than the original OEM partition size: " + strconv.Itoa(oriOemSizeBytes) + " bytes")
	}

	//record the original start sector of the stateful partition
	oriStartSector, err := partutil.ReadPartitionStart(disk, statePartNum)
	if err != nil {
		return err
	}

	//move the stateful partition
	err = partutil.MovePartition(disk, statePartNum, "+"+oemSize)
	if err != nil {
		return err
	}

	//move oem partition to the original start sector of the stateful partition
	err = partutil.MovePartition(disk, oemPartNum, strconv.Itoa(oriStartSector))
	if err != nil {
		return err
	}

	//record the new start sector of the stateful partition
	newStartSector, err := partutil.ReadPartitionStart(disk, statePartNum)
	if err != nil {

		return err
	}

	//extend the oem partition
	err = partutil.ExtendPartition(disk, oemPartNum, newStartSector-1)
	if err != nil {
		return err
	}
	log.Printf("\nCompleted extending OEM partition\n")
	return nil
}
