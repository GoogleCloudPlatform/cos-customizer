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

package partutil

import (
	"cos-customizer/tools/partutil/partutiltest"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"testing"
)

func TestExtendPartition(t *testing.T) {
	var testNames partutiltest.TestNames
	t.Cleanup(func() { partutiltest.TearDown(&testNames) })
	partutiltest.SetupFakeDisk("tmp_disk_extend_partition", "", t, &testNames)

	diskName := testNames.DiskName

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
			"EmptyDiskName",
			"",
			1,
			833,
		},
		// {
		// 	"TooLargeSize",
		// 	diskName,
		// 	1,
		// 	3000,
		// },
	}

	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			if err := ExtendPartition(input.disk, input.partNum, input.end); err == nil {
				t.Fatalf("error not found in test %s", input.testName)
			}
		})
	}

	if err := ExtendPartition(diskName, 1, 833); err != nil {
		t.Fatal("error when extending partition 1 to 833")
	}

	if err := os.Mkdir("./mt", 0777); err != nil {
		t.Fatal("cannot create mount point")
	}
	defer os.Remove("./mt")

	cmdM := fmt.Sprintf("sudo mount %sp1 mt", diskName)
	if err := exec.Command("bash", "-c", cmdM).Run(); err != nil {
		t.Fatalf("error mounting disk file")
	}
	cmdM = "sudo umount mt"
	defer exec.Command("bash", "-c", cmdM).Run()
	cmdD := "df -h | grep mt"
	out, err := exec.Command("bash", "-c", cmdD).Output()
	if err != nil {
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
