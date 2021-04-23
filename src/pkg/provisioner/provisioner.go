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
	"bufio"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"cloud.google.com/go/storage"
	"golang.org/x/sys/unix"
)

//go:embed docker-credential-gcr
var dockerCredentialGCR []byte

// ErrRebootRequired indicates that a reboot is necessary for provisioning to
// continue.
var ErrRebootRequired = errors.New("reboot required to continue provisioning")

// I typically do not like this style of mocking, but I think it's the best
// option in this case. These functions cannot execute at all in a normal test
// environment because they require root privileges. Even if the address to
// mount is owned by the caller, these functions will fail. To take them out of
// the test codepath, we can mock them here.
//
// I have considered an alternative involving writing a unixPkg interface and
// passing it through the Deps struct. But it doesn't give us much for its
// additional verbosity.
var mountFunc = unix.Mount
var unmountFunc = unix.Unmount

func mountOptions(rootDir, mountPoint string) (uintptr, error) {
	mountInfoFile, err := os.Open(filepath.Join(rootDir, "proc/self/mountinfo"))
	if err != nil {
		return 0, err
	}
	defer mountInfoFile.Close()
	scanner := bufio.NewScanner(mountInfoFile)
	var options string
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 6 {
			return 0, fmt.Errorf("invalid line in mountinfo: %q", scanner.Text())
		}
		if fields[4] == mountPoint {
			options = fields[5]
			break
		}
	}
	if options == "" {
		return 0, fmt.Errorf("mountpoint %q not found", mountPoint)
	}
	var parsedOptions uintptr
	for _, opt := range strings.FieldsFunc(options, func(r rune) bool { return r == ',' }) {
		// String representations of mount options are viewable here:
		// https://github.com/torvalds/linux/blob/8404c9fbc84b741f66cff7d4934a25dd2c344452/fs/proc_namespace.c#L66
		//
		// "ro" vs "rw" is special cased:
		// https://github.com/torvalds/linux/blob/8404c9fbc84b741f66cff7d4934a25dd2c344452/fs/proc_namespace.c#L159
		switch opt {
		case "nosuid":
			parsedOptions |= unix.MS_NOSUID
		case "nodev":
			parsedOptions |= unix.MS_NODEV
		case "noexec":
			parsedOptions |= unix.MS_NOEXEC
		case "noatime":
			parsedOptions |= unix.MS_NOATIME
		case "nodiratime":
			parsedOptions |= unix.MS_NODIRATIME
		case "relatime":
			parsedOptions |= unix.MS_RELATIME
		case "nosymfollow":
			parsedOptions |= unix.MS_NOSYMFOLLOW
		case "ro":
			parsedOptions |= unix.MS_RDONLY
		}
	}
	return parsedOptions, nil
}

func setup(runState *state, rootDir string, systemd *systemdClient) error {
	log.Println("Setting up environment...")
	if err := systemd.stop("update-engine.service"); err != nil {
		return err
	}
	if err := mountFunc("tmpfs", filepath.Join(rootDir, "root"), "tmpfs", 0, ""); err != nil {
		return fmt.Errorf("error mounting tmpfs at /root: %v", err)
	}
	binPath := filepath.Join(runState.dir, "bin")
	dockerCredentialGCRPath := filepath.Join(binPath, "docker-credential-gcr")
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		if err := os.Mkdir(binPath, 0744); err != nil {
			return err
		}
	}
	if _, err := os.Stat(dockerCredentialGCRPath); os.IsNotExist(err) {
		if err := ioutil.WriteFile(dockerCredentialGCRPath, dockerCredentialGCR, 0744); err != nil {
			return err
		}
	}
	// docker-credential-gcr will complain if docker-credential-gcr is not in the
	// PATH
	pathVar := os.Getenv("PATH")
	if err := os.Setenv("PATH", binPath+":"+pathVar); err != nil {
		return fmt.Errorf("could not update PATH environment variable: %v", err)
	}
	// Ensure that docker-credential-gcr is on an executable mount
	if err := mountFunc(binPath, binPath, "ext4", unix.MS_BIND, ""); err != nil {
		return fmt.Errorf("error bind mounting %q: %v", dockerCredentialGCRPath, err)
	}
	opts, err := mountOptions(rootDir, binPath)
	if err != nil {
		return err
	}
	if err := mountFunc("", binPath, "", unix.MS_REMOUNT|unix.MS_BIND|opts&^unix.MS_NOEXEC, ""); err != nil {
		return fmt.Errorf("error remounting %q as executable: %v", dockerCredentialGCRPath, err)
	}
	// Run docker-credential-gcr
	cmd := exec.Command(dockerCredentialGCRPath, "configure-docker")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error in cmd docker-credential-gcr, see stderr for details: %v", err)
	}
	log.Println("Done setting up the environment")
	return nil
}

func stopServices(systemd *systemdClient) error {
	log.Println("Stopping services...")
	for _, s := range []string{
		"crash-reporter.service",
		"crash-sender.service",
		"device_policy_manager.service",
		"metrics-daemon.service",
		"update-engine.service",
	} {
		if err := systemd.stop(s); err != nil {
			return err
		}
	}
	log.Println("Done stopping services.")
	return nil
}

func zeroAllFiles(dir string) error {
	if _, err := os.Stat(dir); err != nil && os.IsNotExist(err) {
		return nil
	}
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %q: %v", path, err)
		}
		if info.IsDir() {
			return nil
		}
		// Truncate the file
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
		return nil
	})
}

func cleanupDir(dir string) error {
	fileInfos, err := ioutil.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		} else {
			return err
		}
	}
	for _, fi := range fileInfos {
		if err := os.RemoveAll(filepath.Join(dir, fi.Name())); err != nil {
			return err
		}
	}
	return nil
}

func cleanup(rootDir, stateDir string) error {
	log.Println("Cleaning up machine state...")
	binPath := filepath.Join(stateDir, "bin")
	if err := unmountFunc(binPath, 0); err != nil {
		return fmt.Errorf("error unmounting %q: %v", binPath, err)
	}
	if err := os.RemoveAll(stateDir); err != nil {
		return err
	}
	if err := unmountFunc(filepath.Join(rootDir, "root"), 0); err != nil {
		// This error can be non-fatal because this is cleaning up a tmpfs mount,
		// which doesn't impact the final image output in any way
		log.Printf("Non-fatal error unmounting tmpfs at /root: %v", err)
	}
	if err := os.RemoveAll(filepath.Join(rootDir, "mnt", "stateful_partition", "etc")); err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, d := range []string{
		filepath.Join(rootDir, "var", "cache"),
		filepath.Join(rootDir, "var", "tmp"),
		filepath.Join(rootDir, "var", "lib", "crash_reporter"),
		filepath.Join(rootDir, "var", "lib", "metrics"),
		filepath.Join(rootDir, "var", "lib", "systemd"),
		filepath.Join(rootDir, "var", "lib", "update_engine"),
		filepath.Join(rootDir, "var", "lib", "whitelist"),
	} {
		if err := cleanupDir(d); err != nil {
			return err
		}
	}
	// There are a few files in /var/log that need to exist for daemons to work.
	// The best way to clear logs is to zero them out instead of deleting them.
	if err := zeroAllFiles(filepath.Join(rootDir, "var", "log")); err != nil {
		return err
	}
	log.Println("Done cleaning up machine state")
	return nil
}

func executeSteps(s *state) error {
	for i, step := range s.data.Config.Steps {
		// In the case where executeSteps runs after a reboot, we need to skip
		// through all the steps that have already been completed.
		if i < s.data.CurrentStep {
			continue
		}
		abstractStep, err := parseStep(step.Type, step.Args)
		if err != nil {
			return fmt.Errorf("error parsing step %d: %v", i, err)
		}
		if err := abstractStep.run(s); err != nil {
			return fmt.Errorf("error in step %d: %v", i, err)
		}
		// Persist our most recent completed step to disk, so we can resume after a reboot.
		s.data.CurrentStep++
		if err := s.write(); err != nil {
			return err
		}
	}
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
	// RootdevCmd is the path to the rootdev binary.
	RootdevCmd string
	// CgptCmd is the path to the cgpt binary.
	CgptCmd string
	// Resize2fsCmd is the path to the resize2fs binary.
	Resize2fsCmd string
	// E2fsckCmd is the path to the e2fsck binary.
	E2fsckCmd string
	// RootDir is the path to the root file system. Should be "/" in all real
	// runtime situations.
	RootDir string
}

func run(deps Deps, runState *state) (err error) {
	systemd := &systemdClient{systemctl: deps.SystemctlCmd}
	if err := repartitionBootDisk(deps, runState); err != nil {
		return err
	}
	if err := setup(runState, deps.RootDir, systemd); err != nil {
		return err
	}
	if err := executeSteps(runState); err != nil {
		return err
	}
	if err := stopServices(systemd); err != nil {
		return fmt.Errorf("error stopping services: %v", err)
	}
	if err := cleanup(deps.RootDir, runState.dir); err != nil {
		return fmt.Errorf("error in cleanup: %v", err)
	}
	log.Println("Done provisioning machine")
	return nil
}

// Run runs a full provisioning flow based on the provided config. The stateDir
// is used for persisting data used as part of provisioning. The stateDir allows
// the provisioning flow to be interrupted (e.g. by a reboot) and resumed.
func Run(ctx context.Context, deps Deps, stateDir string, c Config) error {
	log.Println("Provisioning machine...")
	runState, err := initState(ctx, deps, stateDir, c)
	if err != nil {
		return err
	}
	return run(deps, runState)
}

// Resume resumes provisioning from the state provided at stateDir.
func Resume(ctx context.Context, deps Deps, stateDir string) (err error) {
	log.Println("Resuming provisioning...")
	runState, err := loadState(stateDir)
	if err != nil {
		return err
	}
	return run(deps, runState)
}
