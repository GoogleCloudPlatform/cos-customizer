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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"testing"
)

// A file in tools/partutil/testdata is used as the simulation of a disk.
// When a test program starts, it will copy the file and work on it. Its size is 600K. It has three partitions as follows:
// 1.partition 8, OEM partition, 100K
// 2.partition 2, middle partition, 100K
// 3.partition 1, stateful partition, 100K

func TestExtendPartitionFails(t *testing.T) {
	var testNames partutiltest.TestNames
	t.Cleanup(func() { partutiltest.TearDown(&testNames) })
	partutiltest.SetupFakeDisk("tmp_disk_extend_partition_fails", "", t, &testNames)

	diskName := testNames.DiskName
	testData := []struct {
		testName string
		disk     string
		partNum  int
		end      uint64
	}{
		{
			testName: "SameSize",
			disk:     diskName,
			partNum:  1,
			end:      633,
		}, {
			testName: "InvalidDisk",
			disk:     "./testdata/no_disk",
			partNum:  1,
			end:      833,
		}, {
			testName: "InvalidPartition",
			disk:     diskName,
			partNum:  0,
			end:      833,
		}, {
			testName: "NonexistPartition",
			disk:     diskName,
			partNum:  100,
			end:      833,
		}, {
			testName: "SmallerSize",
			disk:     diskName,
			partNum:  100,
			end:      500,
		}, {
			testName: "EmptyDiskName",
			disk:     "",
			partNum:  100,
			end:      833,
		}, {
			testName: "TooLargeSize",
			disk:     diskName,
			partNum:  1,
			end:      3000,
		},
	}

	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			if err := ExtendPartition(input.disk, input.partNum, input.end); err == nil {
				t.Fatalf("error not found in test %s", input.testName)
			}
		})
	}
}

func TestExtendPartitionPasses(t *testing.T) {
	var testNames partutiltest.TestNames
	t.Cleanup(func() { partutiltest.TearDown(&testNames) })
	partutiltest.SetupFakeDisk("tmp_disk_extend_partition_passes", "", t, &testNames)

	diskName := testNames.DiskName

	if err := ExtendPartition(diskName, 1, 833); err != nil {
		t.Fatalf("error when extending partition 1 to 833, error msg: (%v)", err)
	}

	if err := os.Mkdir("./mt", 0777); err != nil {
		t.Fatal("cannot create mount point ./mt")
	}
	defer os.Remove("./mt")

	if err := exec.Command("sudo", "mount", diskName+"p1", "mt").Run(); err != nil {
		t.Fatalf("error mounting disk file, partName: %q, error msg: (%v)", diskName+"p1", err)
	}
	defer exec.Command("sudo", "umount", "mt").Run()

	cmdD := "df -h | grep mt"
	out, err := exec.Command("bash", "-c", cmdD).Output()
	if err != nil {
		t.Fatalf("error reading df -h, error msg: (%v)", err)
	}
	size, err := readSize(string(out))
	if err != nil {
		t.Fatalf("cannot read fs size from df -h, "+
			"df line: %q, error msg: (%v) ", string(out), err)
	}
	if size <= 180 {
		t.Fatalf("wrong fs size of %q, "+
			"actual size: %d, expected size: >180", diskName+"p1", size)
	}
}

// readSize reads fs size from df -h, looking for the first unit K
// to find the size
func readSize(out string) (int, error) {
	pos := 0
	res := -1
	for pos < len(out) && out[pos] != 'K' {
		pos++
	}
	if pos == len(out) {
		return 0, errors.New("cannot find unit K")
	}
	if !(out[pos-3] >= '0' && out[pos-3] <= '9') {
		return 0, errors.New("have less than 3 digits")
	}
	res, err := strconv.Atoi(out[pos-3 : pos])
	if err != nil {
		return 0, fmt.Errorf("cannot convert %q to int", string(out[pos-3:pos]))
	}
	return res, nil
}
