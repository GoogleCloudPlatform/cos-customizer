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
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// ConvertSizeToBytes converts a size string to int unit: bytes.
// It takes a string of number with no unit (sectors), unit B, unit K, unit M, or unit G.
func ConvertSizeToBytes(size string) (uint64, error) {
	const B = 1
	const K = 1024
	const M = K * 1024
	const G = M * 1024
	const Sec = 512

	var err error
	var res uint64 = 0
	l := len(size)

	if l <= 0 {
		return 0, errors.New("invalid oemSize: empty string")
	}

	if size[0] < '0' || size[0] > '9' {
		return 0, fmt.Errorf("invalid oemSize, the first char should be digit, "+
			"input size: %q", size)
	}

	if size[l-1] >= '0' && size[l-1] <= '9' {
		res, err = strconv.ParseUint(size, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("cannot convert %q to int", size)
		}
		res *= Sec
	} else {
		res, err = strconv.ParseUint(size[0:l-1], 10, 64)
		if err != nil {
			return 0, fmt.Errorf("cannot convert %q in input: %q to int", string(size[0:l-1]), size)
		}
		switch size[l-1] {
		case 'B':
			res *= B
		case 'K':
			res *= K
		case 'M':
			res *= M
		case 'G':
			res *= G
		default:
			return 0, fmt.Errorf("wrong format for oemSize, input: %q, "+
				"expecting input like 10G, 200M, 600K, 5000B or 1024", size)
		}
	}

	return res, nil
}

// ConvertSizeToGBRoundUp converts input size to GB unit.
// Rounded up, since extend disk can only take GB unit.
// Used by Daisy workflow to resize the disk.
func ConvertSizeToGBRoundUp(size string) (uint64, error) {
	sizeByte, err := ConvertSizeToBytes(size)
	if err != nil {
		return 0, err
	}
	sizeGB := sizeByte >> 30
	if (sizeGB << 30) != sizeByte {
		sizeGB++
	}
	return sizeGB, nil
}

// PartNumIntToString converts input int partNumInt into string,
// if disk ends with number, add 'p' to the front.
// Example: /dev/loop5p1
func PartNumIntToString(disk string, partNumInt int) (string, error) {
	if len(disk) <= 0 {
		return "", errors.New("empty disk name")
	}
	partNum := strconv.Itoa(partNumInt)
	if disk[len(disk)-1] >= '0' && disk[len(disk)-1] <= '9' {
		partNum = "p" + partNum
	}
	return partNum, nil
}

// GetPartUUID finds the PartUUID of a partition using blkid
func GetPartUUID(partName string) (string, error) {
	var idBuf bytes.Buffer
	cmd := exec.Command("sudo", "blkid")
	cmd.Stdout = &idBuf
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("error in running blkid, "+
			"std output:%s, error msg: (%v)", idBuf.String(), err)
	}
	// blkid has output like:
	// /dev/sda1: LABEL="STATE" UUID="120991ff-4f12-43bf-b962-17325185121d" TYPE="ext4"
	// /dev/sda3: LABEL="ROOT-A" SEC_TYPE="ext2" TYPE="ext4" PARTLABEL="ROOT-A" PARTUUID="00ce255b-db42-1e47-a62b-735c7a9a7397"
	// /dev/sda8: LABEL="OEM" UUID="1401457b-449d-4755-9a1e-57054b287489" TYPE="ext4" PARTLABEL="OEM" PARTUUID="9db2ae75-98dc-5b4f-a38b-b3cb0b80b17f"
	// /dev/sda12: SEC_TYPE="msdos" LABEL="EFI-SYSTEM" UUID="F6E7-003C" TYPE="vfat" PARTLABEL="EFI-SYSTEM" PARTUUID="aaea6e5e-bc5f-2542-b19a-66c2daa4d5a8"
	// /dev/dm-0: LABEL="ROOT-A" SEC_TYPE="ext2" TYPE="ext4"
	// /dev/sda2: PARTLABEL="KERN-A" PARTUUID="de4778dd-c187-8343-b86c-e122f9d234c0"
	// /dev/sda4: PARTLABEL="KERN-B" PARTUUID="7b8374db-78b2-2748-bab9-a52d0867455b"
	// /dev/sda5: PARTLABEL="ROOT-B" PARTUUID="8ac60384-1187-9e49-91ce-3abd8da295a7"
	// /dev/sda11: PARTLABEL="RWFW" PARTUUID="682ef1a5-f7f6-7d42-a407-5d8ad0430fc1"
	lines := strings.Split(idBuf.String(), "\n")
	for _, line := range lines {
		if !strings.HasPrefix(line, partName+":") {
			continue
		}
		for _, content := range strings.Fields(line) {
			if !strings.HasPrefix(content, "PARTUUID") {
				continue
			}
			return strings.Trim(strings.Split(content, "=")[1], "\""), nil
		}
	}
	return "", fmt.Errorf("partition UUID not found, input: partName=%q ,"+
		"output of \"blkid\": %s", partName, idBuf.String())
}

// FindLast4KSector returns the last 4K bytes aligned sector from start.
// If input is a 4K aligned sector, return itself.
func FindLast4KSector(start uint64) uint64 {
	var mask uint64 = 7
	return start & (^mask)
}
