package partutil

import (
	"io"
	"os"
	"os/exec"
	"testing"
)

func setupFakeDisk(copyName, srcPrefix string, t *testing.T) (string, string) {
	copyFile := ""
	diskName := ""
	src, err := os.Open("./" + srcPrefix + "disk_file/ori_disk")
	if err != nil {
		t.Fatal("cannot open ori_disk")
	}
	defer src.Close()

	copyFile = "./" + srcPrefix + "disk_file/" + copyName
	dest, err := os.Create(copyFile)
	if err != nil {
		t.Fatal("cannot create tmp disk file")
	}
	defer os.Remove("./disk_file/" + copyName)
	_, err = io.Copy(dest, src)
	if err != nil {
		t.Fatal("error copying disk file")
	}
	dest.Close()

	cmd := "sudo losetup -fP --show ./disk_file/" + copyName
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		t.Fatal("error losetup disk file")
	}
	diskName = string(out)
	diskName = diskName[:len(diskName)-1]

	return copyFile, diskName
}

func tearDown(copyFile, diskName string) {
	if diskName != "" {
		cmd := "sudo losetup -d " + diskName
		exec.Command("bash", "-c", cmd).Run()
	}
	if copyFile != "" {
		os.Remove(copyFile)
	}
}

func TestMovePartition(t *testing.T) {
	diskName := ""
	copyFile := ""
	t.Cleanup(func() { tearDown(copyFile, diskName) })
	copyFile, diskName = setupFakeDisk("tmp_disk_move_partition", "", t)

	type testStruct struct {
		testName string
		disk     string
		partNum  int
		dest     string
	}

	testData := []testStruct{
		{
			"MoveByDistanceTooSmall",
			diskName,
			1,
			"-500K",
		}, {
			"MoveByDistanceTooLarge",
			diskName,
			1,
			"+500K",
		}, {
			"InvalidDisk",
			"./disk_file/no_disk",
			8,
			"+100K",
		}, {
			"InvalidPartition",
			diskName,
			0,
			"+100K",
		}, {
			"NonexistPartition",
			diskName,
			100,
			"+100K",
		}, {
			"MoveToInvalidPosSmall",
			diskName,
			1,
			"0",
		}, {
			"MoveToInvalidPosLarge",
			diskName,
			1,
			"5000",
		}, {
			"MoveCollision",
			diskName,
			8,
			"300",
		}, {
			"EmptyDiskName",
			"",
			1,
			"+100K",
		},
	}

	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			err := MovePartition(input.disk, input.partNum, input.dest)
			if err == nil {
				t.Fatalf("error not found in test %s", input.testName)
			}
		})
	}
	step := testStruct{
		"MoveByDistancePos",
		diskName,
		1,
		"+150K",
	}
	err := MovePartition(step.disk, step.partNum, step.dest)
	if err != nil {
		t.Fatalf("error in test %s", step.testName)
	}
	step = testStruct{
		"MoveByDistanceNeg",
		diskName,
		1,
		"-40K",
	}
	err = MovePartition(step.disk, step.partNum, step.dest)
	if err != nil {
		t.Fatalf("error in test %s", step.testName)
	}
	step = testStruct{
		"MoveToPos",
		diskName,
		8,
		"434",
	}
	err = MovePartition(step.disk, step.partNum, step.dest)
	if err != nil {
		t.Fatalf("error in test %s", step.testName)
	}

	pos, err := ReadPartitionStart(step.disk, 8)
	if err != nil {
		t.Fatal("cannot read partition start")
	}
	if pos != 434 {
		t.Fatal("error result position for partition 8")
	}
	pos, err = ReadPartitionStart(step.disk, 1)
	if err != nil {
		t.Fatal("cannot read partition start")
	}
	if pos != 654 {
		t.Fatal("error result position for partition 8")
	}
}
