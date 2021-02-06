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
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

// MountEFIPartition mounts the EFI partition (/dev/sda12)
// and returns the path where grub.cfg is at.
func MountEFIPartition() (string, error) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", fmt.Errorf("error in creating tempDir, "+
			"error msg: (%v)", err)
	}
	cmd := exec.Command("sudo", "mount", "/dev/sda12", dir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error in mounting /dev/sda12 at %q, "+
			"error msg: (%v)", dir, err)
	}
	return dir + "/efi/boot/grub.cfg", nil
}

// UnmountEFIPartition unmounts the EFI partition (/dev/sda12)
func UnmountEFIPartition() error {
	cmd := exec.Command("sudo", "umount", "/dev/sda12")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error in unmounting /dev/sda12"+
			"error msg: (%v)", err)
	}
	return nil
}

// GRUBContains seaches for the command string inside of the GRUB file
func GRUBContains(grubPath, cmd string) (bool, error) {
	grubContent, err := ioutil.ReadFile(grubPath)
	if err != nil {
		return false, fmt.Errorf("cannot read grub.cfg at %q, "+
			"input: grubPath=%q, cmd=%q, error msg:(%v)", grubPath, grubPath, cmd, err)
	}
	if strings.Contains(string(grubContent), cmd) {
		return true, nil
	}
	return false, nil
}

// AddCmdToGRUB adds a command string to
// after every `cros_efi` command
func AddCmdToGRUB(grubPath, cmd string) error {
	const appendPoint = "cros_efi"
	const appendOffset = len(appendPoint)
	grubContent, err := ioutil.ReadFile(grubPath)
	if err != nil {
		return fmt.Errorf("cannot read grub.cfg at %q, "+
			"input: grubPath=%q, cmd=%q, error msg:(%v)", grubPath, grubPath, cmd, err)
	}
	lines := strings.Split(string(grubContent), "\n")
	for idx, line := range lines {
		if !strings.Contains(line, appendPoint) {
			continue
		}
		startPos := strings.Index(line, appendPoint) + appendOffset
		lines[idx] = fmt.Sprintf("%s %s %s", line[:startPos], cmd, line[startPos:])
	}
	// new content of grub.cfg
	grubContent = []byte(strings.Join(lines, "\n"))
	err = ioutil.WriteFile(grubPath, grubContent, 0755)
	if err != nil {
		return fmt.Errorf("cannot write to grub.cfg at %q, "+
			"input: grubPath=%q, cmd=%q, error msg:(%v)", grubPath, grubPath, cmd, err)
	}
	return nil
}
