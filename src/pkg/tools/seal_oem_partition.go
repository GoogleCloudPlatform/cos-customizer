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
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/tools/partutil"
)

// SealOEMPartition sets the hashtree of the OEM partition
// with "veritysetup" and modifies the kernel command line to
// verify the OEM partition at boot time.
func SealOEMPartition(veritysetupImgPath string, oemFSSize4K uint64) error {
	const devName = "oemroot"
	imageID, err := loadVeritysetupImage(veritysetupImgPath)
	if err != nil {
		return fmt.Errorf("cannot load veritysetup image at %q, error msg:(%v)", veritysetupImgPath, err)
	}
	log.Println("docker image for veritysetup loaded.")
	if err := unmountOEMPartition(); err != nil {
		return fmt.Errorf("cannot umount OEM partition, error msg:(%v)", err)
	}
	log.Println("OEM partition unmounted.")
	hash, salt, err := veritysetup(imageID, oemFSSize4K)
	if err != nil {
		return fmt.Errorf("cannot run veritysetup, input:oemFSSize4K=%d, "+
			"error msg:(%v)", oemFSSize4K, err)
	}
	grubPath, err := partutil.MountEFIPartition()
	log.Println("EFI partition mounted.")
	if err != nil {
		return fmt.Errorf("cannot mount EFI partition (/dev/sda12), error msg:(%v)", err)
	}
	defer partutil.UnmountEFIPartition()
	partUUID, err := partutil.GetPartUUID("/dev/sda8")
	if err != nil {
		return fmt.Errorf("cannot read partUUID of /dev/sda8")
	}
	if err := appendDMEntryToGRUB(grubPath, devName, partUUID, hash, salt, oemFSSize4K); err != nil {
		return fmt.Errorf("error in appending entry to grub.cfg, input:oemFSSize4K=%d, "+
			"error msg:(%v)", oemFSSize4K, err)
	}
	log.Println("kernel command line modified.")
	if err := removeVeritysetupImage(imageID); err != nil {
		return fmt.Errorf("cannot remove veritysetup image, error msg:(%v)", err)
	}
	log.Println("docker image for veritysetup removed.")
	log.Println("OEM partition sealed.")
	return nil
}

// loadVeritysetupImage loads the docker image of veritysetup.
// return the image ID.
func loadVeritysetupImage(imgPath string) (string, error) {
	cmd := exec.Command("sudo", "docker", "load", "-i", imgPath)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error in loading docker image, "+
			"input: imgPath=%q, error msg: (%v)", imgPath, err)
	}
	var idBuf bytes.Buffer
	cmd = exec.Command("sudo", "docker", "images", "veritysetup:veritysetup", "-q")
	cmd.Stdout = &idBuf
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error in reading image ID, "+
			"cmd:%q,std output:%s, error msg: (%v)",
			"sudo docker images veritysetup:veritysetup -q", idBuf.String(), err)
	}
	if idBuf.Len() == 0 {
		return "", fmt.Errorf("image ID not found, "+
			"input: imgPath=%q", imgPath)
	}
	imageID := idBuf.String()
	return imageID[:len(imageID)-1], nil
}

// removeVeritysetupImage removes the container and docker image of veritysetup
func removeVeritysetupImage(imageID string) error {
	cmd := exec.Command("sudo", "docker", "rmi", imageID)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error in removing docker image, "+
			"id=%q, error msg: (%v)", imageID, err)
	}
	return nil
}

// unmountOEMPartition checks whether the OEM partititon (/dev/sda8)
// is mounted, if so, unmount it.
func unmountOEMPartition() error {
	var buf bytes.Buffer
	cmd := exec.Command("df")
	cmd.Stdout = &buf
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error in running df, "+
			"std output:%s, error msg: (%v)", buf.String(), err)
	}
	if !strings.Contains(buf.String(), "/dev/sda8") {
		return nil
	}
	cmd = exec.Command("sudo", "umount", "/dev/sda8")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error in unmounting /dev/sda8, "+
			"error msg: (%v)", err)
	}
	return nil
}

// veritysetup runs the docker container command veritysetup to build hash tree of OEM partition
// and generate hash root value and salt value.
func veritysetup(imageID string, oemFSSize4K uint64) (string, string, error) {
	dataBlocks := "--data-blocks=" + strconv.FormatUint(oemFSSize4K, 10)
	// --hash-offset is in Bytes
	hashOffset := "--hash-offset=" + strconv.FormatUint(oemFSSize4K<<12, 10)
	cmd := exec.Command("sudo", "docker", "run", "--rm", "--name", "veritysetup", "--privileged",
		"-v", "/dev:/dev", imageID, "veritysetup", "format", "/dev/sda8", "/dev/sda8",
		"--data-block-size=4096", "--hash-block-size=4096", dataBlocks, hashOffset,
		"--no-superblock", "--format=0")
	var verityBuf bytes.Buffer
	cmd.Stdout = &verityBuf
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("error in running docker veritysetup, "+
			"input: oemFSSize4K=%d, std output:%s,error msg: (%v)",
			oemFSSize4K, verityBuf.String(), err)
	}
	// Output of veritysetup is like:
	// VERITY header information for /dev/sdb1
	// UUID:
	// Hash type:              0
	// Data blocks:            2048
	// Data block size:        4096
	// Hash block size:        4096
	// Hash algorithm:         sha256
	// Salt:                   9cd7ba29a1771b2097a7d72be8c13b29766d7617c3b924eb0cf23ff5071fee47
	// Root hash:              d6b862d01e01e6417a1b5e7eb0eed2a2189594b74325dd0749cd83bbf78f5dc8
	hash := ""
	salt := ""
	for _, line := range strings.Split(verityBuf.String(), "\n") {
		if strings.HasPrefix(line, "Root hash:") {
			hash = strings.TrimSpace(strings.Split(line, ":")[1])
		} else if strings.HasPrefix(line, "Salt:") {
			salt = strings.TrimSpace(strings.Split(line, ":")[1])
		}
	}
	if hash == "" || salt == "" {
		return "", "", fmt.Errorf("error in veritsetup output format, cannot find \"Salt:\" or \"Root hash:\", "+
			"input: oemFSSize4K=%d, veritysetup output: %s", oemFSSize4K, verityBuf.String())
	}
	return hash, salt, nil
}

// appendDMEntryToGRUB appends an dm-verity table entry to kernel command line in grub.cfg
// A target line in grub.cfg looks like
// ...... root=/dev/dm-0 dm="1 vroot none ro 1,0 4077568 verity
// payload=PARTUUID=8AC60384-1187-9E49-91CE-3ABD8DA295A7
// hashtree=PARTUUID=8AC60384-1187-9E49-91CE-3ABD8DA295A7 hashstart=4077568 alg=sha256
// root_hexdigest=xxxxxxxx salt=xxxxxxxx"
func appendDMEntryToGRUB(grubPath, name, partUUID, hash, salt string, oemFSSize4K uint64) error {
	// from 4K blocks to 512B sectors
	oemFSSizeSector := oemFSSize4K << 3
	entryString := fmt.Sprintf("%s none ro 1, 0 %d verity payload=PARTUUID=%s hashtree=PARTUUID=%s "+
		"hashstart=%d alg=sha256 root_hexdigest=%s salt=%s\"", name, oemFSSizeSector,
		partUUID, partUUID, oemFSSizeSector, hash, salt)
	grubContent, err := ioutil.ReadFile(grubPath)
	if err != nil {
		return fmt.Errorf("cannot read grub.cfg at %q, "+
			"input: grubPath=%q, name=%q, partUUID=%q, oemFSSize4K=%d, hash=%q, salt=%q, "+
			"error msg:(%v)", grubPath, grubPath, name, partUUID, oemFSSize4K, hash, salt, err)
	}
	lines := strings.Split(string(grubContent), "\n")
	// add the entry to all kernel command lines containing "dm="
	for idx, line := range lines {
		if !strings.Contains(line, "dm=") {
			continue
		}
		startPos := strings.Index(line, "dm=")
		// remove the end quote.
		lineBuf := []rune(line[:len(line)-1])
		// add number of entries.
		lineBuf[startPos+4] = '2'
		lines[idx] = strings.Join(append(strings.Split(string(lineBuf), ","), entryString), ",")
	}
	// new content of grub.cfg
	grubContent = []byte(strings.Join(lines, "\n"))
	err = ioutil.WriteFile(grubPath, grubContent, 0755)
	if err != nil {
		return fmt.Errorf("cannot write to grub.cfg at %q, "+
			"input: grubPath=%q, name=%q, partUUID=%q, oemFSSize4K=%d, hash=%q, salt=%q, "+
			"error msg:(%v)", grubPath, grubPath, name, partUUID, oemFSSize4K, hash, salt, err)
	}
	return nil
}
