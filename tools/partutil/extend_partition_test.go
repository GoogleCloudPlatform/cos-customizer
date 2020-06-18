package partutil

import (
	"log"
	"os/exec"
	"strconv"
	"testing"
)

func TestExtendPartition(t *testing.T) {
	diskName := ""
	copyFile := ""
	t.Cleanup(func() { tearDown(copyFile, diskName) })
	copyFile, diskName = setupFakeDisk("tmp_disk_extend_partition", "", t)

	testData := []struct {
		testName string
		disk     string
		partNum  int
		end      int
	}{
		{
			"SameSize",
			diskName,
			1,
			633,
		}, {
			"InvalidDisk",
			"./disk_file/no_disk",
			8,
			833,
		}, {
			"InvalidPartition",
			diskName,
			0,
			833,
		}, {
			"NonexistPartition",
			diskName,
			100,
			833,
		}, {
			"SmallerSize",
			diskName,
			1,
			500,
		}, {
			"TooLargeSize",
			diskName,
			1,
			3000,
		}, {
			"EmptyDiskName",
			"",
			1,
			833,
		},
	}

	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			err := ExtendPartition(input.disk, input.partNum, input.end)
			if err == nil {
				t.Fatalf("error not found in test %s", input.testName)
			}
		})
	}

	err := ExtendPartition(diskName, 1, 833)
	if err != nil {
		t.Fatal("error when extending partition 1 to 833")
	}
	cmdM := "sudo mount " + diskName + "p1 mt"
	err = exec.Command("bash", "-c", cmdM).Run()
	if err != nil {
		log.Println("CMMMMMMM: " + cmdM)

		t.Fatal("error mounting disk file")
	}
	cmdM = "sudo umount mt"
	defer exec.Command("bash", "-c", cmdM).Run()
	cmdD := "df -h | grep mt"
	out, err := exec.Command("bash", "-c", cmdD).Output()
	if err != nil {
		log.Println("CMDDDDDDD: " + cmdD)
		t.Fatal("error reading df")
	}
	if readSize(string(out)) <= 180 {
		t.Fatalf("wrong file system size of partition 1\n INFO: %s", string(out))
	}
}

func readSize(out string) int {
	pos := 0
	res := -1
	for pos < len(out) && out[pos] != 'K' {
		pos++
	}
	if pos == len(out) {
		return -1
	}
	if !(out[pos-3] >= '0' && out[pos-3] <= '9') {
		return -1
	}
	res, err := strconv.Atoi(out[pos-3 : pos])
	if err != nil {
		return -1
	}
	return res
}
