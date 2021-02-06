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
	"fmt"
	"log"

	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/tools/partutil"
)

// DisableSystemdService disables the auto-update service.
func DisableSystemdService(service string) error {
	cmd := "systemd.mask=" + service
	grubPath, err := partutil.MountEFIPartition()
	if err != nil {
		return fmt.Errorf("cannot mount EFI partition,"+
			"error msg:(%v)", err)
	}
	defer partutil.UnmountEFIPartition()
	contains, err := partutil.GRUBContains(grubPath, cmd)
	if err != nil {
		return fmt.Errorf("cannot read GRUB file at %q,"+
			"error msg:(%v)", grubPath, err)
	}
	if contains {
		return nil
	}
	if err := partutil.AddCmdToGRUB(grubPath, cmd); err != nil {
		return fmt.Errorf("cannot add commmand to GRUB file at %q,"+
			"cmd=%q, error msg:(%v)", grubPath, cmd, err)
	}
	log.Printf("%q service disabled.", service)
	return nil
}
