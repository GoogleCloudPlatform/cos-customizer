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

// Package provisioner exports behaviors for provisioning COS systems
// end-to-end. These behaviors are intended to run on a COS system.
package provisioner

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"

	"cloud.google.com/go/storage"
	"golang.org/x/sys/unix"
)

func setup(dockerCredentialGCR string, systemd *systemdClient) error {
	log.Println("Setting up environment...")
	if err := systemd.stop("update-engine.service"); err != nil {
		return err
	}
	if err := unix.Mount("", "/root", "tmpfs", 0, ""); err != nil {
		return fmt.Errorf("error mounting tmpfs at /root: %v", err)
	}
	cmd := exec.Command(dockerCredentialGCR, "configure-docker")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error in cmd docker-credential-gcr, see stderr for details: %v", err)
	}
	log.Println("Done setting up the environment")
	return nil
}

func cleanup(stateDir string) error {
	log.Println("Cleaning up machine state...")
	if err := os.RemoveAll(stateDir); err != nil {
		return err
	}
	log.Println("Done cleaning up machine state")
	return nil
}

// Deps contains provisioner service dependencies.
type Deps struct {
	// GCSClient is used to access Google Cloud Storage.
	GCSClient *storage.Client
	// TarCmd is used for tar.
	TarCmd string
	// SystemctlCmd is used to access systemd.
	SystemctlCmd string
	// DockerCredentialGCR is the path to the docker-credential-gcr binary.
	DockerCredentialGCR string
}

// Run runs a full provisioning flow based on the provided config. The stateDir
// is used for persisting data used as part of provisioning. The stateDir allows
// the provisioning flow to be interrupted (e.g. by a reboot) and resumed.
func Run(ctx context.Context, deps Deps, stateDir string, c Config) (err error) {
	log.Println("Provisioning machine...")
	systemd := &systemdClient{systemctl: deps.SystemctlCmd}
	if _, err := initState(ctx, deps, stateDir, c); err != nil {
		return err
	}
	if err := setup(deps.DockerCredentialGCR, systemd); err != nil {
		return err
	}
	// TODO(rkolchmeyer): Implement the actual provisioning behavior
	if err := cleanup(stateDir); err != nil {
		return fmt.Errorf("error in cleanup: %v", err)
	}
	log.Println("Done provisioning machine")
	return nil
}
