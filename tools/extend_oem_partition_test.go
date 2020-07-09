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
	"errors"
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

func TestExtendOEMPartitionFails(t *testing.T) {
	var testNames partutiltest.TestNames
	t.Cleanup(func() { partutiltest.TearDown(&testNames) })
	partutiltest.SetupFakeDisk("tmp_disk_extend_oem_partition_fails", "partutil/", t, &testNames)

	diskName := testNames.DiskName

	testData := []struct {
		testName     string
		disk         string
		statePartNum int
		oemPartNum   int
		size         string
	}{
		{
			testName:     "InvalidDisk",
			disk:         "./partutil/testdata/no_disk",
			statePartNum: 1,
			oemPartNum:   8,
			size:         "200K",
		}, {
			testName:     "InvalidStatePartition",
			disk:         diskName,
			statePartNum: 100,
			oemPartNum:   8,
			size:         "200K",
		}, {
			testName:     "InvalidOEMPartition",
			disk:         diskName,
			statePartNum: 1,
			oemPartNum:   800,
			size:         "200K",
		}, {
			testName:     "InvalidSize1",
			disk:         diskName,
			statePartNum: 1,
			oemPartNum:   8,
			size:         "-200K",
		}, {
			testName:     "InvalidSize2",
			disk:         diskName,
			statePartNum: 1,
			oemPartNum:   8,
			size:         "200T",
		}, {
			testName:     "InvalidSize3",
			disk:         diskName,
			statePartNum: 1,
			oemPartNum:   8,
			size:         "A45M",
		}, {
			testName:     "InvalidSize4",
			disk:         diskName,
			statePartNum: 1,
			oemPartNum:   8,
			size:         "+200K",
		}, {
			testName:     "InvalidSize5",
			disk:         diskName,
			statePartNum: 1,
			oemPartNum:   8,
			size:         "",
		}, {
			testName:     "TooLarge",
			disk:         diskName,
			statePartNum: 1,
			oemPartNum:   8,
			size:         "800M",
		}, {
			testName:     "EmptyDiskName",
			disk:         "",
			statePartNum: 1,
			oemPartNum:   8,
			size:         "200K",
		},
	}

	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			if err := ExtendOEMPartition(input.disk, input.statePartNum, input.oemPartNum, input.size); err == nil {
				t.Fatalf("error not found in test %s", input.testName)
			}
		})
	}
}

func TestExtendOEMPartitionWarnings(t *testing.T) {
	var testNames partutiltest.TestNames
	t.Cleanup(func() { partutiltest.TearDown(&testNames) })
	partutiltest.SetupFakeDisk("tmp_disk_extend_oem_partition_warnings", "partutil/", t, &testNames)

	diskName := testNames.DiskName

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
			"SameSize",
			diskName,
			1,
			8,
			"100K",
		},
	}

	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			if err := ExtendOEMPartition(input.disk, input.statePartNum, input.oemPartNum, input.size); err != nil {
				t.Fatalf("error in test %s, error msg: (%v)", input.testName, err)
			}
		})
	}
}

func TestExtendOEMPartitionPasses(t *testing.T) {
	var testNames partutiltest.TestNames
	t.Cleanup(func() { partutiltest.TearDown(&testNames) })
	partutiltest.SetupFakeDisk("tmp_disk_extend_oem_partition_passes", "partutil/", t, &testNames)

	diskName := testNames.DiskName

	if err := ExtendOEMPartition(diskName, 1, 8, "200K"); err != nil {
		t.Fatalf("error when extending OEM partition, error msg: (%v)", err)
	}

	if err := os.Mkdir("./mt", 0777); err != nil {
		t.Fatalf("cannot create mount point, error msg: (%v)", err)
	}
	defer os.Remove("./mt")

	testData := []struct {
		partitionName string
		wantContent   string
		wantSize      int
	}{
		{
			partitionName: diskName + "p8",
			wantContent:   "This is partition 8 OEM partition",
			wantSize:      180,
		},
		{
			partitionName: diskName + "p1",
			wantContent:   "This is partition 1 stateful partition",
			wantSize:      80,
		},
		{
			partitionName: diskName + "p2",
			wantContent:   "This is partition 2 middle partition",
			wantSize:      80,
		},
	}

	// since need to mount at the same dir, tests need to be executed sequentially
	for _, input := range testData {
		mountAndCheck(input.partitionName, input.wantContent, t, input.wantSize)
	}
}

// readSize reads partition fs size from df -h
// a line looks like:
// tmpfs           100K     0  100K   0% /dev/lxd
// (Filesystem      Size  Used Avail Use% Mounted on)
func readSize(out string) (int, error) {
	pos := 0
	res := -1
	var err error
	for pos < len(out) && out[pos] != 'K' {
		pos++
	}
	if pos == len(out) {
		return -1, errors.New("cannot find unit K")
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

// mountAndCheck mounts a partition and check its fs size and content of a file in it
func mountAndCheck(partName, wantLine string, t *testing.T, size int) {
	t.Helper()
	if err := exec.Command("sudo", "mount", partName, "mt").Run(); err != nil {
		t.Fatalf("error mounting %q, error msg: (%v)", partName, err)
	}
	defer exec.Command("sudo", "umount", "mt").Run()
	cmdD := "df -h | grep mt"
	out, err := exec.Command("bash", "-c", cmdD).Output()
	if err != nil {
		t.Fatalf("error reading df -h of %q, error msg: (%v)", partName, err)
	}
	if len(out) <= 0 {
		t.Fatalf("cannot find partition %q", partName)
	}
	oldSize, err := readSize(string(out))
	if err != nil {
		t.Fatalf("cannot read fs size, the line in df -h: %q, error msg: (%v)", string(out), err)
	}
	if oldSize <= size {
		t.Fatalf("wrong file system size of partition, fs info: %q, expected: %d", string(out), size)
	}

	f, err := os.Open("mt/content")
	if err != nil {
		t.Fatalf("cannot open content file in %q, error msg: (%v)", partName, err)
	}
	defer f.Close()
	rd := bufio.NewReader(f)
	line, _, err := rd.ReadLine()
	if err != nil {
		t.Fatalf("cannot ReadLine in %q, error msg: (%v)", partName, err)
	}
	if string(line) != wantLine {
		t.Fatalf("content in %q is corrupted, actual line: %q, wanted line: %q", partName, string(line), wantLine)
	}
}
