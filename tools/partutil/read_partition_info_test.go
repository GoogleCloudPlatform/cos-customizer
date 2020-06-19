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

func TestReadPartitionSize(t *testing.T) {
	var testNames partutiltest.TestNames
	t.Cleanup(func() { partutiltest.TearDown(&testNames) })
	partutiltest.SetupFakeDisk("tmp_disk_extend_partition", "", t, &testNames)

	diskName := testNames.DiskName

	testData := []struct {
		testName string
		disk     string
		partNum  int
		want     int
		expErr   bool
	}{
		{
			"100KPart",
			diskName,
			8,
			200,
			false,
		}, {
			"InvalidDisk",
			"./disk_file/no_disk",
			8,
			0,
			true,
		}, {
			"InvalidPartition",
			diskName,
			0,
			0,
			true,
		}, {
			"NonexistPartition",
			diskName,
			100,
			0,
			true,
		}, {
			"EmptyDiskName",
			"",
			1,
			0,
			true,
		},
	}

	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			res, err := ReadPartitionSize(input.disk, input.partNum)
			if (err != nil) != input.expErr {
				if input.expErr {
					t.Fatalf("error not found in test %s", input.testName)
				} else {
					t.Fatalf("error in test %s", input.testName)
				}
			}
			if !input.expErr {
				if res != input.want {
					t.Fatalf("wrong result: %s %d to %d, exp: %d", input.disk, input.partNum, res, input.want)
				}
			}
		})
	}
}

func TestReadPartitionStart(t *testing.T) {
	var testNames partutiltest.TestNames
	t.Cleanup(func() { partutiltest.TearDown(&testNames) })
	partutiltest.SetupFakeDisk("tmp_disk_extend_partition", "", t, &testNames)

	diskName := testNames.DiskName

	testData := []struct {
		testName string
		disk     string
		partNum  int
		want     int
		expErr   bool
	}{
		{
			"100KPart",
			diskName,
			8,
			200,
			false,
		}, {
			"InvalidDisk",
			"./disk_file/no_disk",
			8,
			0,
			true,
		}, {
			"InvalidPartition",
			diskName,
			0,
			0,
			true,
		}, {
			"NonexistPartition",
			diskName,
			1000,
			0,
			true,
		},
	}

	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			res, err := ReadPartitionSize(input.disk, input.partNum)
			if (err != nil) != input.expErr {
				if input.expErr {
					t.Fatalf("error not found in test %s", input.testName)
				} else {
					t.Fatalf("error in test %s", input.testName)
				}
			}
			if !input.expErr {
				if res != input.want {
					t.Fatalf("wrong result: %s %d to %d, exp: %d", input.disk, input.partNum, res, input.want)
				}
			}
		})
	}
}
