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
// Disk ori_disk: 600 KiB, 614400 bytes, 1200 sectors
// Units: sectors of 1 * 512 = 512 bytes
// Sector size (logical/physical): 512 bytes / 512 bytes
// I/O size (minimum/optimal): 512 bytes / 512 bytes
// Disklabel type: gpt
// Disk identifier: 9CEB1C17-FCD7-8F4F-ADE7-097A2DB2F996

// Device     Start   End Sectors  Size Type
// ori_disk1    434   633     200  100K Linux filesystem
// ori_disk2    234   433     200  100K Linux filesystem
// ori_disk8     34   233     200  100K Linux filesystemskk

// Partition table entries are not in disk order.

func TestParsePartitionTableFails(t *testing.T) {

	testData := struct {
		testName string
		table    string
		partName string
	}{

		testName: "NoPartitionFound",
		table:    "abc",
		partName: "sda1",
	}

	if _, err := ParsePartitionTable(testData.table, testData.partName, false, func(p *PartContent) {}); err == nil {
		t.Fatalf("error not found in %s", testData.testName)
	}
}

func TestParsePartitionTablePasses(t *testing.T) {

	testData := struct {
		testName string
		table    string
		partName string
		start    uint64
		size     uint64
		want     string
	}{

		testName: "ValidChange",
		table:    "/dev/sdb11 : start=     4401152, size=     2097152, type=0FC63DAF-8483-4772-8E79-3D69D8477DE4, uuid=3B41256B-E064-544A-9101-D2647C0B3A38\n/dev/sdb1 : start=     6498304, size=      204800, type=0FC63DAF-8483-4772-8E79-3D69D8477DE4, uuid=9479C34A-49A6-9442-A56F-956396DFAC20\n",
		partName: "/dev/sdb1",
		start:    5001,
		size:     4096,
		want:     "/dev/sdb11 : start=     4401152, size=     2097152, type=0FC63DAF-8483-4772-8E79-3D69D8477DE4, uuid=3B41256B-E064-544A-9101-D2647C0B3A38\n/dev/sdb1 : start=5001, size=4096, type=0FC63DAF-8483-4772-8E79-3D69D8477DE4, uuid=9479C34A-49A6-9442-A56F-956396DFAC20\n",
	}

	res, err := ParsePartitionTable(testData.table, testData.partName, true, func(p *PartContent) {
		p.Start = testData.start
		p.Size = testData.size
	})
	if err != nil {
		t.Fatalf("error found in %s, error msg: (%v)", testData.testName, err)
	}
	if res != testData.want {
		t.Fatalf("wrong result in %q, res: %q, expected: %q", testData.testName, res, testData.want)

	}

}

func TestReadPartitionSizeFails(t *testing.T) {
	var testNames partutiltest.TestNames
	t.Cleanup(func() { partutiltest.TearDown(&testNames) })
	partutiltest.SetupFakeDisk("tmp_disk_read_partition_size_fails", "", t, &testNames)

	diskName := testNames.DiskName

	testData := []struct {
		testName string
		disk     string
		partNum  int
	}{{
		testName: "InvalidDisk",
		disk:     "./testdata/no_disk",
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
	partutiltest.SetupFakeDisk("tmp_disk_read_partition_size_passes", "", t, &testNames)

	diskName := testNames.DiskName

	input := struct {
		testName string
		disk     string
		partNum  int
		want     uint64
	}{
		testName: "200KPart",
		disk:     diskName,
		partNum:  8,
		want:     200,
	}

	res, err := ReadPartitionSize(input.disk, input.partNum)
	if err != nil {

	}
	if res != input.want {
		t.Fatalf("wrong result: %q partition %d at %d, exp: %d", input.disk, input.partNum, res, input.want)
	}
}

func TestReadPartitionStartFails(t *testing.T) {
	var testNames partutiltest.TestNames
	t.Cleanup(func() { partutiltest.TearDown(&testNames) })
	partutiltest.SetupFakeDisk("tmp_disk_read_partition_start_fails", "", t, &testNames)

	diskName := testNames.DiskName

	testData := []struct {
		testName string
		disk     string
		partNum  int
	}{
		{
			testName: "InvalidDisk",
			disk:     "./testdata/no_disk",
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
	partutiltest.SetupFakeDisk("tmp_disk_read_partition_start_passes", "", t, &testNames)

	diskName := testNames.DiskName

	input := struct {
		testName string
		disk     string
		partNum  int
		want     uint64
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
