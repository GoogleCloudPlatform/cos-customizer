package partutil

import (
	"testing"
)

func TestReadPartitionSize(t *testing.T) {
	diskName := ""
	copyFile := ""
	t.Cleanup(func() { tearDown(copyFile, diskName) })
	copyFile, diskName = setupFakeDisk("tmp_disk_read_size", "", t)

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
