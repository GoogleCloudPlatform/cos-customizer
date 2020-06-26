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
	"testing"
)

// A file in tools/partutil/disk_file is used as the simulation of a disk.
// When a test program starts, it will copy the file and work on it. Its size is 600K. It has three partitions as follows:
// 1.partition 8, OEM partition, 100K
// 2.partition 2, middle partition, 100K
// 3.partition 1, stateful partition, 100K

func TestReadPartitionSizeFails(t *testing.T) {
	var testNames partutiltest.TestNames
	t.Cleanup(func() { partutiltest.TearDown(&testNames) })
	partutiltest.SetupFakeDisk("tmp_disk_extend_partition", "", t, &testNames)

	diskName := testNames.DiskName

	testData := []struct {
		testName string
		disk     string
		partNum  int
	}{{
		testName: "InvalidDisk",
		disk:     "./disk_file/no_disk",
		partNum:  8,
	}, {
		testName: "InvalidPartition",
		disk:     diskName,
		partNum:  0,
	}, {
		testName: "NonexistPartition",
		disk:     diskName,
		partNum:  100,
	}, {
		testName: "EmptyDiskName",
		disk:     "",
		partNum:  1,
	},
	}

	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			_, err := ReadPartitionSize(input.disk, input.partNum)
			if err == nil {
				t.Fatalf("error not found in test %s", input.testName)
			}
		})
	}
}

func TestReadPartitionSizePasses(t *testing.T) {
	var testNames partutiltest.TestNames
	t.Cleanup(func() { partutiltest.TearDown(&testNames) })
	partutiltest.SetupFakeDisk("tmp_disk_extend_partition", "", t, &testNames)

	diskName := testNames.DiskName

	input := struct {
		testName string
		disk     string
		partNum  int
		want     int
	}{
		testName: "200KPart",
		disk:     diskName,
		partNum:  8,
		want:     200,
	}

	res, err := ReadPartitionSize(input.disk, input.partNum)
	if err != nil {

	}
	if res != 200 {
		t.Fatalf("wrong result: %s partition %d at %d, exp: %d", input.disk, input.partNum, res, input.want)
	}
}

func TestReadPartitionStartFails(t *testing.T) {
	var testNames partutiltest.TestNames
	t.Cleanup(func() { partutiltest.TearDown(&testNames) })
	partutiltest.SetupFakeDisk("tmp_disk_extend_partition", "", t, &testNames)

	diskName := testNames.DiskName

	testData := []struct {
		testName string
		disk     string
		partNum  int
	}{
		{
			testName: "InvalidDisk",
			disk:     "./disk_file/no_disk",
			partNum:  8,
		}, {
			testName: "InvalidPartition",
			disk:     diskName,
			partNum:  0,
		}, {
			testName: "NonexistPartition",
			disk:     diskName,
			partNum:  1000,
		},
	}

	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			_, err := ReadPartitionStart(input.disk, input.partNum)
			if err == nil {
				t.Fatalf("error not found in test %s", input.testName)
			}

		})
	}
}

func TestReadPartitionStartPasses(t *testing.T) {
	var testNames partutiltest.TestNames
	t.Cleanup(func() { partutiltest.TearDown(&testNames) })
	partutiltest.SetupFakeDisk("tmp_disk_extend_partition", "", t, &testNames)

	diskName := testNames.DiskName

	input := struct {
		testName string
		disk     string
		partNum  int
		want     int
	}{
		testName: "PartStartAt434",
		disk:     diskName,
		partNum:  1,
		want:     434,
	}

	start, err := ReadPartitionStart(input.disk, input.partNum)
	if err != nil {
		t.Fatalf("error in test %s, error msg: (%v)", input.testName, err)
	}
	if start != input.want {
		t.Fatalf("wrong result in test %s, start: %d, expected: %d", input.testName, start, input.want)
	}
}
