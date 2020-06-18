package partutil

import (
	"errors"
	"os/exec"
	"strconv"
	"strings"
)

//ReadPartitionStart reads the start sector of a partition
func ReadPartitionStart(disk string, partNumInt int) (int, error) {
	if len(disk) <= 0 || partNumInt <= 0 {
		return 0, errors.New("empty input for disk name or partition number")
	}

	//for cases like loop5p1
	partNum := strconv.Itoa(partNumInt)
	if disk[len(disk)-1] >= '0' && disk[len(disk)-1] <= '9' {
		partNum = "p" + partNum
	}

	//dump partition table and grep the line
	partName := disk + partNum
	cmd := string("sudo sfdisk --dump ") + disk + " |grep " + partName
	line, err := exec.Command("/bin/bash", "-c", cmd).Output()
	if Check(err, cmd) {
		return -1, err
	}
	if len(line) < 4 { //not find a valid info line
		return -1, errors.New("cannot find partition " + partName)
	}
	start := -1
	ls := strings.Split(string(line), " ")
	mode := 0
	for _, word := range ls {
		switch mode {
		case 0: //looking for start sector
			if word == "start=" {
				mode = 1
			}
		case 1:
			if len(word) > 0 { //a valid sector number has at least 1 digit
				mode = 2
				start, err = strconv.Atoi(word[:len(word)-1]) //a comma at the end
				if Check(err, "cannot covert start sector to int") {
					return 0, err
				}
			}
		default:
			return -1, errors.New("Error: error in looking for partition")
		}
		if mode == 2 {
			break
		}
	}
	if start == -1 {
		return -1, errors.New("Error: error in looking for partition")
	}
	return start, nil
}
