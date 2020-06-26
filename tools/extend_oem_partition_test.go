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
	"bufio"
	"cos-customizer/tools/partutil/partutiltest"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"testing"
)

func TestExtendOEMPartition(t *testing.T) {
	var testNames partutiltest.TestNames
	t.Cleanup(func() { partutiltest.TearDown(&testNames) })
	partutiltest.SetupFakeDisk("tmp_disk_extend_oem_partition", "partutil/", t, &testNames)

	diskName := testNames.DiskName

	testData := []struct {
		testName     string
		disk         string
		statePartNum int
		oemPartNum   int
		size         string
	}{
		{
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
			"InvalidOEMPartition",
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
			if err := ExtendOEMPartition(input.disk, input.statePartNum, input.oemPartNum, input.size); err == nil {
				t.Fatalf("error not found in test %s", input.testName)
			}
		})
	}

	testData2 := []struct {
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
			"SameSize",
			diskName,
			1,
			8,
			"100K",
		},
	}

	for _, input := range testData2 {
		t.Run(input.testName, func(t *testing.T) {
			if err := ExtendOEMPartition(input.disk, input.statePartNum, input.oemPartNum, input.size); err != nil {
				t.Fatalf("error in test %s", input.testName)
			}
		})
	}

	if err := ExtendOEMPartition(diskName, 1, 8, "200K"); err != nil {
		t.Fatal("error when extending OEM partition")
	}
	if err := os.Mkdir("./mt", 0777); err != nil {
		t.Fatal("cannot create mount point")
	}
	defer os.Remove("./mt")
	mountAndCheck(diskName+"p8", "This is partition 8 OEM partition", t, 180)
	mountAndCheck(diskName+"p1", "This is partition 1 stateful partition", t, 80)
	mountAndCheck(diskName+"p2", "This is partition 2 middle partition", t, 80)
}

// read partition fs size from df -h
// a line looks like:
// tmpfs           100K     0  100K   0% /dev/lxd
// (Filesystem      Size  Used Avail Use% Mounted on)
func readSize(out string) (int, error) {
	pos := 0
	res := -1
	var err error
	for out[pos] != 'K' {
		pos++
	}

	if out[pos-3] != ' ' {
		res, err = strconv.Atoi(out[pos-3 : pos])
		if err != nil {
			return -1, err
		}
	} else {
		res, err = strconv.Atoi(out[pos-2 : pos])
		if err != nil {
			return -1, err
		}
	}

	return res, nil
}

func mountAndCheck(partName, wantLine string, t *testing.T, size int) {
	cmdM := fmt.Sprintf("sudo mount %s mt", partName)
	if err := exec.Command("bash", "-c", cmdM).Run(); err != nil {
		t.Fatalf("error mounting %s", partName)
	}
	cmdM = "sudo umount mt"
	defer exec.Command("bash", "-c", cmdM).Run()
	cmdD := "df -h | grep mt"
	out, err := exec.Command("bash", "-c", cmdD).Output()
	if err != nil {
		t.Fatalf("error reading df %s", partName)
	}
	oldSize, err := readSize(string(out))
	if err != nil {
		t.Fatalf("cannot read fs size, the line in df -h: %s", string(out))
	}
	if oldSize <= size {
		t.Fatalf("wrong file system size of partition \n INFO: %s", string(out))
	}

	f, err := os.Open("mt/content")
	if err != nil {
		t.Fatalf("cannot open file in %s", partName)
	}
	defer f.Close()
	rd := bufio.NewReader(f)
	line, _, err := rd.ReadLine()
	if err != nil {
		t.Fatal("cannot ReadLine in p8")
	}
	if string(line) != wantLine {
		t.Fatalf("content in %s corrupted", partName)
	}
}
