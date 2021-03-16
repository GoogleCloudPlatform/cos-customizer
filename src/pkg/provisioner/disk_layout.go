// Copyright 2021 Google LLC
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

package provisioner

import (
	_ "embed"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"

	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/tools/partutil"
	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/utils"
)

//go:embed _handle_disk_layout.bin
var handleDiskLayoutBin []byte

func switchRoot(deps Deps, runState *state) (err error) {
	if !runState.data.Config.BootDisk.ReclaimSDA3 {
		log.Println("ReclaimSDA3 is not set, not switching root device")
		return nil
	}
	sda3Device := filepath.Join(deps.RootDir, "dev", "sda3")
	sda5Device := filepath.Join(deps.RootDir, "dev", "sda5")
	rootDev, err := exec.Command(deps.RootdevCmd, "-s").Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("error running rootdev: %v: stderr = %q", exitErr, string(exitErr.Stderr))
		}
		return fmt.Errorf("error running rootdev: %v", err)
	}
	if strings.TrimSpace(string(rootDev)) == sda5Device {
		log.Println("Current root device is /dev/sda5, not switching root device")
		return nil
	}
	log.Println("Need to switch root device")
	log.Println("Copying sda3 to sda5...")
	in, err := os.Open(sda3Device)
	if err != nil {
		return err
	}
	defer utils.CheckClose(in, "error closing /dev/sda3", &err)
	out, err := os.Create(sda5Device)
	if err != nil {
		return err
	}
	defer utils.CheckClose(out, "error closing /dev/sda5", &err)
	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("error copying from sda3 to sda5: %v", err)
	}
	log.Println("Setting GPT priority...")
	device := filepath.Join(deps.RootDir, "dev", "sda")
	if err := utils.RunCommand([]string{deps.CgptCmd, "prioritize", "-P", "5", "-i", "4", device}, "", nil); err != nil {
		return err
	}
	log.Println("Reboot required to switch root device")
	return ErrRebootRequired
}

func shrinkSDA3(deps Deps, runState *state) error {
	if !runState.data.Config.BootDisk.ReclaimSDA3 {
		log.Println("ReclaimSDA3 is not set, not shrinking sda3")
		return nil
	}
	device := filepath.Join(deps.RootDir, "dev", "sda")
	minimal, err := partutil.IsPartitionMinimal(device, 3)
	if err != nil {
		return fmt.Errorf("error checking /dev/sda3 size: %v", err)
	}
	if minimal {
		log.Println("/dev/sda3 is minimally sized, not shrinking sda3")
		return nil
	}
	log.Println("ReclaimSDA3 is set, and /dev/sda3 is not minimal; now shrinking sda3")
	if _, err := partutil.MinimizePartition(device, 3); err != nil {
		return fmt.Errorf("error minimizing /dev/sda3: %v", err)
	}
	log.Println("Reboot required to reload partition table changes")
	return ErrRebootRequired
}

func setupOnShutdownUnit(deps Deps, runState *state) (err error) {
	if err := mountFunc("", filepath.Join(deps.RootDir, "tmp"), "", unix.MS_REMOUNT|unix.MS_NOSUID|unix.MS_NODEV, ""); err != nil {
		return fmt.Errorf("error remounting /tmp as exec: %v", err)
	}
	if err := ioutil.WriteFile(filepath.Join(deps.RootDir, "tmp", "handle_disk_layout.bin"), handleDiskLayoutBin, 0744); err != nil {
		return err
	}
	data := fmt.Sprintf(`[Unit]
Description=Run after everything unmounted
DefaultDependencies=false
Conflicts=shutdown.target
Before=mnt-stateful_partition.mount usr-share-oem.mount
After=tmp.mount

[Service]
Type=oneshot
RemainAfterExit=true
ExecStart=/bin/true
ExecStop=/bin/bash -c '/tmp/handle_disk_layout.bin /dev/sda 1 8 "%s" "%t" 2>&1 | sed "s/^/BuildStatus: /"'
TimeoutStopSec=600
StandardOutput=tty
StandardError=tty
TTYPath=/dev/ttyS2
`, runState.data.Config.BootDisk.OEMSize, runState.data.Config.BootDisk.ReclaimSDA3)
	if err := ioutil.WriteFile(filepath.Join(deps.RootDir, "etc/systemd/system/last-run.service"), []byte(data), 0664); err != nil {
		return err
	}
	systemd := systemdClient{systemctl: deps.SystemctlCmd}
	if err := systemd.start("last-run.service", []string{"--no-block"}); err != nil {
		return err
	}
	// journald needs to be stopped in order for the stateful partition to be
	// unmounted at shutdown. We need the stateful partition to be unmounted so
	// that disk repartitioning can occur.
	if err := systemd.stopJournald(deps.RootDir); err != nil {
		return err
	}
	return nil
}

func calcSDA3End(device string) (uint64, error) {
	sda3Start, err := partutil.ReadPartitionStart(device, 3)
	if err != nil {
		return 0, err
	}
	sda3Size, err := partutil.ReadPartitionSize(device, 3)
	if err != nil {
		return 0, err
	}
	sda3End := sda3Start + sda3Size - 1
	return sda3End, nil
}

func waitForDiskResize(deps Deps, runState *state) error {
	if !runState.data.Config.BootDisk.WaitForDiskResize {
		log.Println("WaitForDiskResize is not set, not waiting for a boot disk resize")
		return nil
	}
	if runState.data.DiskResizeComplete {
		log.Println("Already finished waiting for disk resize, not waiting again")
		return nil
	}
	startSize, err := ioutil.ReadFile(filepath.Join(deps.RootDir, "sys/class/block/sda/size"))
	if err != nil {
		return err
	}
	log.Println("WaitForDiskResize is set; waiting for the boot disk size to change. Timeout is 3 minutes")
	start := time.Now()
	end := start.Add(3 * time.Minute)
	for time.Now().Before(end) {
		curSize, err := ioutil.ReadFile(filepath.Join(deps.RootDir, "sys/class/block/sda/size"))
		if err != nil {
			return err
		}
		if string(curSize) != string(startSize) {
			log.Printf("Boot disk size has changed: start %q, end %q", strings.TrimSpace(string(startSize)), strings.TrimSpace(string(curSize)))
			runState.data.DiskResizeComplete = true
			return runState.write()
		}
		time.Sleep(time.Second)
	}
	return errors.New("timed out waiting for disk resize")
}

func relocatePartitions(deps Deps, runState *state) error {
	if !runState.data.Config.BootDisk.ReclaimSDA3 && runState.data.Config.BootDisk.OEMSize == "" {
		log.Println("ReclaimSDA3 is not set, OEM resize not requested, not relocating partitions")
		return nil
	}
	device := filepath.Join(deps.RootDir, "dev", "sda")
	if runState.data.Config.BootDisk.OEMSize != "" {
		// Check if OEM partition is after sda3; if so, then we're done
		oemStart, err := partutil.ReadPartitionStart(device, 8)
		if err != nil {
			return err
		}
		sda3End, err := calcSDA3End(device)
		if err != nil {
			return err
		}
		if oemStart > sda3End {
			log.Println("OEM resize requested, OEM appears to be relocated after sda3. Partition relocation is complete")
			return nil
		}
	} else {
		// Check two things:
		// 1. sda3 is minimal
		// 2. Stateful partition is located immediately after sda3
		//
		// If both are true, we are done.
		minimal, err := partutil.IsPartitionMinimal(device, 3)
		if err != nil {
			return err
		}
		statefulStart, err := partutil.ReadPartitionStart(device, 1)
		if err != nil {
			return err
		}
		sda3Start, err := partutil.ReadPartitionStart(device, 3)
		// The stateful partition is relocated 4096 sectors after the start of sda3.
		// See src/pkg/tools/handle_disk_layout.go for details.
		if minimal && statefulStart == sda3Start+4096 {
			log.Println("ReclaimSDA3 is set, sda3 appears to have been reclaimed. Partition relocation is complete")
			return nil
		}
	}
	// Partition relocation must be done. Prepare for disk relocation to happen on
	// the next reboot
	log.Println("Partition relocation is required. Preparing for partition relocation to occur on the next reboot")
	if err := setupOnShutdownUnit(deps, runState); err != nil {
		return err
	}
	log.Println("Reboot required to relocate partitions")
	return ErrRebootRequired
}

func resizeOEMFileSystem(deps Deps, runState *state) error {
	if !runState.data.Config.BootDisk.ReclaimSDA3 && runState.data.Config.BootDisk.OEMSize == "" {
		log.Println("ReclaimSDA3 is not set, OEM resize not requested, partition relocation did not occur, FS resize unnecessary")
		return nil
	}
	// Check if OEM partition is after sda3; if so, then relocation occurred and
	// we need to resize the file system.
	device := filepath.Join(deps.RootDir, "dev", "sda")
	sda3End, err := calcSDA3End(device)
	if err != nil {
		return err
	}
	oemStart, err := partutil.ReadPartitionStart(device, 8)
	if err != nil {
		return err
	}
	if oemStart < sda3End {
		log.Println("OEM partition is before sda3; relocation did not occur, FS resize unnecessary")
		return nil
	}
	log.Println("Partition relocation appears to have occurred, resizing the OEM file system")
	systemd := systemdClient{systemctl: deps.SystemctlCmd}
	if err := systemd.stop("usr-share-oem.mount"); err != nil {
		return err
	}
	sda8 := filepath.Join(deps.RootDir, "dev", "sda8")
	if err := utils.RunCommand([]string{deps.E2fsckCmd, "-fp", sda8}, "", nil); err != nil {
		return err
	}
	resizeArgs := []string{deps.Resize2fsCmd, sda8}
	if runState.data.Config.BootDisk.OEMFSSize4K != 0 {
		resizeArgs = append(resizeArgs, strconv.FormatUint(runState.data.Config.BootDisk.OEMFSSize4K, 10))
	}
	if err := utils.RunCommand(resizeArgs, "", nil); err != nil {
		return err
	}
	if err := systemd.start("usr-share-oem.mount", nil); err != nil {
		return err
	}
	log.Println("OEM file system resized to account for available space")
	return nil
}

// repartitionBootDisk executes all behaviors related to repartitioning the boot
// disk. Most of these behaviors require a reboot. To keep reboots simple (e.g.
// we don't want to initiate a reboot when deferred statements are unresolved),
// we handle reboots by returning ErrRebootRequired and asking the caller to
// initiate the reboot.
func repartitionBootDisk(deps Deps, runState *state) error {
	if err := switchRoot(deps, runState); err != nil {
		return err
	}
	if err := shrinkSDA3(deps, runState); err != nil {
		return err
	}
	if err := waitForDiskResize(deps, runState); err != nil {
		return err
	}
	if err := relocatePartitions(deps, runState); err != nil {
		return err
	}
	if err := resizeOEMFileSystem(deps, runState); err != nil {
		return err
	}
	return nil
}
