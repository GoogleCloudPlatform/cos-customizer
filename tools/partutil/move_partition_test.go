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

// A file in tools/partutil/testdata is used as the simulation of a disk.
// When a test program starts, it will copy the file and work on it. Its size is 600K. It has three partitions as follows:
// 1.partition 8, OEM partition, 100K
// 2.partition 2, middle partition, 100K
// 3.partition 1, stateful partition, 100K

func TestMovePartitionFails(t *testing.T) {
	var testNames partutiltest.TestNames
	t.Cleanup(func() { partutiltest.TearDown(&testNames) })
	partutiltest.SetupFakeDisk("tmp_disk_move_partition_fails", "", t, &testNames)

	diskName := testNames.DiskName

	testData := []struct {
		testName string
		disk     string
		partNum  int
		dest     string
	}{
		{
			testName: "MoveByDistanceTooSmall",
			disk:     diskName,
			partNum:  1,
			dest:     "-500K",
		}, {
			testName: "MoveByDistanceTooLarge",
			disk:     diskName,
			partNum:  1,
			dest:     "+500K",
		}, {
			testName: "InvalidDisk",
			disk:     "./testdata/no_disk",
			partNum:  8,
			dest:     "+100K",
		}, {
			testName: "InvalidPartition",
			disk:     diskName,
			partNum:  0,
			dest:     "+100K",
		}, {
			testName: "NonexistPartition",
			disk:     diskName,
			partNum:  100,
			dest:     "+100K",
		}, {
			testName: "MoveToInvalidPosSmall",
			disk:     diskName,
			partNum:  1,
			dest:     "0",
		}, {
			testName: "MoveToInvalidPosLarge",
			disk:     diskName,
			partNum:  1,
			dest:     "5000",
		}, {
			testName: "MoveCollision",
			disk:     diskName,
			partNum:  8,
			dest:     "300",
		}, {
			testName: "EmptyDiskName",
			disk:     "",
			partNum:  1,
			dest:     "+100K",
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
}

func TestMovePartitionPasses(t *testing.T) {
	var testNames partutiltest.TestNames
	t.Cleanup(func() { partutiltest.TearDown(&testNames) })
	partutiltest.SetupFakeDisk("tmp_disk_move_partition_passes", "", t, &testNames)

	diskName := testNames.DiskName

	if err := MovePartition(diskName, 1, "+150K"); err != nil {
		t.Fatalf("error in test MovePartitionByDistancePos, error msg: (%v)", err)
	}

	testData := []struct {
		testName string
		disk     string
		partNum  int
		dest     string
		want     uint64
	}{{
		testName: "MovePartitionByDistanceNeg",
		disk:     diskName,
		partNum:  1,
		dest:     "-40K",
		want:     654,
	}, {
		testName: "MovePartitionToPosition",
		disk:     diskName,
		partNum:  8,
		dest:     "434",
		want:     434,
	},
	}

	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			err := MovePartition(input.disk, input.partNum, input.dest)
			if err != nil {
				t.Fatalf("error in test %s, error msg: (%v)", input.testName, err)
			}
			pos, err := ReadPartitionStart(input.disk, input.partNum)
			if err != nil {
				t.Fatalf("cannot read partition start of %q partition %d "+
					"error msg: (%v)", input.disk, input.partNum, err)
			}
			if pos != input.want {
				t.Fatalf("error result in test %s, pos: %d, expected: %d",
					input.testName, pos, input.want)
			}
		})
	}
}
