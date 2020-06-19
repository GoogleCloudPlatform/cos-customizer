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

package partutiltest

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"testing"
)

// TestNames is used for testing environment setup and teardown.
type TestNames struct {
	CopyFile string
	DiskName string
}

// SetupFakeDisk copys a file to simulate the disk and work on the copy for tests.
func SetupFakeDisk(copyName, srcPrefix string, t *testing.T, testNames *TestNames) {
	src, err := os.Open(fmt.Sprintf("./%sdisk_file/ori_disk", srcPrefix))
	if err != nil {
		t.Fatal("cannot open ori_disk")
	}
	defer src.Close()

	copyFile := fmt.Sprintf("./%sdisk_file/%s", srcPrefix, copyName)
	testNames.CopyFile = copyFile
	dest, err := os.Create(copyFile)
	if err != nil {
		t.Fatal("cannot create tmp disk file")
	}

	if _, err := io.Copy(dest, src); err != nil {
		t.Fatal("error copying disk file")
	}
	dest.Close()

	cmd := fmt.Sprintf("sudo losetup -fP --show /%s", copyFile)
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		t.Fatal("error losetup disk file")
	}
	diskName := string(out)
	testNames.DiskName = diskName[:len(diskName)-1]
}

// TearDown delete the loop device and the copied file for testing environment.
func TearDown(testNames *TestNames) {
	if testNames.DiskName != "" {
		cmd := fmt.Sprintf("sudo losetup -d %s", testNames.DiskName)
		exec.Command("bash", "-c", cmd).Run()
	}
	if testNames.CopyFile != "" {
		os.Remove(testNames.CopyFile)
	}
}
