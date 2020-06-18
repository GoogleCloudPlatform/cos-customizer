package tools

import (
	"bufio"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
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

	cmd := "sudo losetup -fP --show " + copyFile
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

func TestExtendOemPartition(t *testing.T) {
	diskName := ""
	copyFile := ""
	t.Cleanup(func() { tearDown(copyFile, diskName) })
	copyFile, diskName = setupFakeDisk("tmp_disk_extend_oem_partition", "partutil/", t)

	testData := []struct {
		testName     string
		disk         string
		statePartNum int
		oemPartNum   int
		size         string
	}{
		{
			"SmallerSize",
			diskName,
			1,
			8,
			"60K",
		}, {
			"InvalidDisk",
			"./partutil/disk_file/no_disk",
			1,
			8,
			"200K",
		}, {
			"InvalidStatePartition",
			diskName,
			100,
			8,
			"200K",
		}, {
			"InvalidOemPartition",
			diskName,
			1,
			800,
			"200K",
		}, {
			"InvalidSize1",
			diskName,
			1,
			8,
			"-200K",
		}, {
			"InvalidSize2",
			diskName,
			1,
			8,
			"200T",
		}, {
			"InvalidSize3",
			diskName,
			1,
			8,
			"A45M",
		}, {
			"InvalidSize4",
			diskName,
			1,
			8,
			"+200K",
		}, {
			"InvalidSize5",
			diskName,
			1,
			8,
			"",
		}, {
			"TooLarge",
			diskName,
			1,
			8,
			"800M",
		}, {
			"EmptyDiskName",
			"",
			1,
			8,
			"200K",
		},
	}

	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			err := ExtendOemPartition(input.disk, input.statePartNum, input.oemPartNum, input.size)
			if err == nil {
				t.Fatalf("error not found in test %s", input.testName)
			}
		})
	}

	err := ExtendOemPartition(diskName, 1, 8, "200K")
	if err != nil {
		t.Fatal("error when extending OEM partition")
	}
	mountAndCheck(diskName, "p8", "This is partition 8 OEM partition", t, 180)
	mountAndCheck(diskName, "p1", "This is partition 1 stateful partition", t, 80)
	mountAndCheck(diskName, "p2", "This is partition 2 middle partition", t, 80)
}

func readSize(out string) int {
	pos := 0
	res := -1
	var err error
	for out[pos] != 'K' {
		pos++
	}

	if out[pos-3] != ' ' {
		res, err = strconv.Atoi(out[pos-3 : pos])
		if err != nil {
			return -1
		}
	} else {
		res, err = strconv.Atoi(out[pos-2 : pos])
		if err != nil {
			return -1
		}
	}

	return res
}

func mountAndCheck(diskName, partNum, wantLine string, t *testing.T, size int) {
	cmdM := "sudo mount " + diskName + partNum + " partutil/mt"
	err := exec.Command("bash", "-c", cmdM).Run()
	if err != nil {
		log.Println("CMMMMMMM: " + cmdM)

		t.Fatal("error mounting " + diskName + partNum)
	}
	cmdM = "sudo umount partutil/mt"
	defer exec.Command("bash", "-c", cmdM).Run()
	cmdD := "df -h | grep partutil/mt"
	out, err := exec.Command("bash", "-c", cmdD).Output()
	if err != nil {
		log.Println("CMDDDDDDD: " + cmdD)
		t.Error("error reading df" + diskName + partNum)
	}
	if readSize(string(out)) <= size {
		t.Errorf("wrong file system size of partition \n INFO: %s", string(out))
	}

	f, err := os.Open("partutil/mt/content")
	if err != nil {
		t.Error("cannot open file in " + diskName + partNum)
	}
	defer f.Close()
	rd := bufio.NewReader(f)
	line, _, err := rd.ReadLine()
	if err != nil {
		t.Error("cannot ReadLine in p8")
	}
	if string(line) != wantLine {
		t.Errorf("content in %s corrupted", diskName+partNum)
	}
}
