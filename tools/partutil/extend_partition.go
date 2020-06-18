package partutil

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
)

//ExtendPartition extends a partition to a specific end sector
func ExtendPartition(disk string, partNumInt, end int) error {
	if len(disk) <= 0 || partNumInt <= 0 {
		return errors.New("invalid disk name or partition number")
	}

	//for cases like loop5p1
	partNum := strconv.Itoa(partNumInt)
	if disk[len(disk)-1] >= '0' && disk[len(disk)-1] <= '9' {
		partNum = "p" + partNum
	}

	partName := disk + partNum

	//dump partition table to a file
	cmd := string("sudo sfdisk --dump ") + disk + " > extend_partition_tmp"
	err := ExecCmdToStdout(cmd)
	if Check(err, cmd) {
		return err
	}
	defer os.Remove("extend_partition_tmp")

	err = editPartitionTableFile("extend_partition_tmp", partName, end)
	if Check(err, "editing partition table file of "+partName+" to ending sector at: "+strconv.Itoa(end)) {
		return err
	}

	//write partition table back
	cmd = "sudo sfdisk " + disk + " < " + " extend_partition_tmp "
	err = ExecCmdToStdout(cmd)
	if Check(err, cmd) {
		return err
	}
	log.Printf("\nCompleted extending %s\n\n", partName)

	// check and repair file system in the partition
	cmd = "sudo e2fsck -fp " + partName
	err = ExecCmdToStdout(cmd)
	if Check(err, cmd) {
		return err
	}
	log.Printf("\nCompleted checking file system of %s\n\n", partName)

	// resize file system in the partition
	cmd = "sudo resize2fs " + partName
	err = ExecCmdToStdout(cmd)
	if Check(err, cmd) {
		return err
	}

	log.Printf("\nCompleted updating file system of %s\n\n", partName)
	return nil
}

//change partition table file to extend partition
func editPartitionTableFile(fileName, partName string, end int) error {
	in, err := ioutil.ReadFile(fileName)
	if Check(err, "cannot read partition table file") {
		return err
	}

	lines := strings.Split(string(in), "\n")
	have := false //whether has valid information about the partition
	for i, line := range lines {
		if strings.Contains(line, partName) {
			ls := strings.Split(line, " ")
			mode := 0
			start := -1
			for j, word := range ls {
				switch mode {
				case 0: //looking for start sector
					if word == "start=" {
						mode = 1
					}
				case 1:
					if len(word) > 3 { //a valid sector number has at least 4 digits
						mode = 2
						start, err = strconv.Atoi(word[:len(word)-1]) //a comma at the end
						if Check(err, "cannot convert start sector to int") {
							return err
						}
					}
				case 2:
					if word == "size=" {
						mode = 3
					}
				case 3:
					if len(word) > 3 { //a valid sector number has at least 4 digits

						size, err := strconv.Atoi(word[:len(word)-1]) //a comma at the end
						if Check(err, "cannot convert size to int") {
							return err
						}
						if end-start+1 <= size {
							return errors.New("Error: new size is not larger than the original size")
						}
						have = true //Modification completed
						ls[j] = strconv.Itoa(end+1-start) + ","
					}
				default:
					return errors.New("Error: error in looking for partition")
				}
				if have {
					break
				}
			}

			//recreate the line
			if have {
				lines[i] = strings.Join(ls, " ")
			}
			break
		}
	}
	if !have {
		return errors.New("Error: Partition not found")
	}
	//recreate the partition table file
	changed := strings.Join(lines, "\n")
	err = ioutil.WriteFile(fileName, []byte(changed), 0644)
	if Check(err, "cannot write to partition table file") {
		return err
	}
	return nil
}
