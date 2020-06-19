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

func TestMovePartition(t *testing.T) {
	var testNames partutiltest.TestNames
	t.Cleanup(func() { partutiltest.TearDown(&testNames) })
	partutiltest.SetupFakeDisk("tmp_disk_extend_partition", "", t, &testNames)

	diskName := testNames.DiskName

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

	if err := MovePartition(step.disk, step.partNum, step.dest); err != nil {
		t.Fatalf("error in test %s", step.testName)
	}
	step = testStruct{
		"MoveByDistanceNeg",
		diskName,
		1,
		"-40K",
	}

	if err := MovePartition(step.disk, step.partNum, step.dest); err != nil {
		t.Fatalf("error in test %s", step.testName)
	}
	step = testStruct{
		"MoveToPos",
		diskName,
		8,
		"434",
	}
	if err := MovePartition(step.disk, step.partNum, step.dest); err != nil {
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
