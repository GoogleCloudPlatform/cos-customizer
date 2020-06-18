package partutil

import (
	"errors"
	"log"
	"strconv"
)

//MovePartition move a partition to a start sector
// +XX(sector/G/M/K) or -XX(sector/G/M/K)
//for now it takes disk name like /dev/sda
//partition number like 2
//destination like 2048, +5G or -200M
func MovePartition(disk string, partNumInt int, dest string) error {
	if len(disk) <= 0 || partNumInt <= 0 {
		return errors.New("Error: empty disk name or partition number")
	}
	partNum := strconv.Itoa(partNumInt)

	cmd := "echo " + dest + " | sudo sfdisk --move-data " + disk + " -N " + partNum
	err := ExecCmdToStdout(cmd)
	if Check(err, cmd) {
		return err
	}
	log.Printf("\nCompleted moving %s \n\n", (disk + partNum))
	return nil
}
