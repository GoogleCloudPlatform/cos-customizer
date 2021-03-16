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
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/utils"
)

type InstallGPUStep struct {
	NvidiaDriverVersion      string
	NvidiaDriverMD5Sum       string
	NvidiaInstallDirHost     string
	NvidiaInstallerContainer string
	GCSDepsPrefix            string
}

func (s *InstallGPUStep) validate() error {
	if s.NvidiaDriverVersion == "" {
		return errors.New("invalid args: NvidiaDriverVersion is required in InstallGPU")
	}
	if s.NvidiaInstallerContainer == "" {
		return errors.New("invalid args: NvidiaInstallerContainer is required in InstallGPU")
	}
	return nil
}

func (s *InstallGPUStep) setDefaults() {
	if s.NvidiaInstallDirHost == "" {
		s.NvidiaInstallDirHost = "/var/lib/nvidia"
	}
}

func (s *InstallGPUStep) installScript(path, driverVersion string) (err error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0744)
	if err != nil {
		return err
	}
	defer utils.CheckClose(f, fmt.Sprintf("error closing %q", path), &err)
	t, err := template.New("gpu-script").Parse(gpuSetupScriptTemplate)
	if err != nil {
		return err
	}
	if err := t.Execute(f, &InstallGPUStep{
		NvidiaDriverVersion:      utils.QuoteForShell(driverVersion),
		NvidiaDriverMD5Sum:       utils.QuoteForShell(s.NvidiaDriverMD5Sum),
		NvidiaInstallDirHost:     utils.QuoteForShell(s.NvidiaInstallDirHost),
		NvidiaInstallerContainer: utils.QuoteForShell(s.NvidiaInstallerContainer),
	}); err != nil {
		return fmt.Errorf("error installing %q: %v", path, err)
	}
	return nil
}

func (s *InstallGPUStep) runInstaller(path string) error {
	var downloadURL string
	if s.GCSDepsPrefix != "" {
		downloadURL = "https://storage.googleapis.com/" + strings.TrimPrefix(s.GCSDepsPrefix, "gs://")
	}
	var gpuInstallerDownloadURL string
	if strings.HasSuffix(s.NvidiaDriverVersion, ".run") && downloadURL != "" {
		gpuInstallerDownloadURL = downloadURL + "/" + s.NvidiaDriverVersion
	}
	if err := utils.RunCommand([]string{"/bin/bash", path}, "", append(os.Environ(), []string{
		"COS_DOWNLOAD_GCS=" + downloadURL,
		"GPU_INSTALLER_DOWNLOAD_URL=" + gpuInstallerDownloadURL,
	}...)); err != nil {
		return err
	}
	return nil
}

func (s *InstallGPUStep) run(runState *state) error {
	if err := s.validate(); err != nil {
		return err
	}
	s.setDefaults()
	var driverVersion string
	if strings.HasSuffix(s.NvidiaDriverVersion, ".run") {
		// NVIDIA-Linux-x86_64-450.51.06.run -> 450.51.06
		fields := strings.FieldsFunc(strings.TrimSuffix(s.NvidiaDriverVersion, ".run"), func(r rune) bool { return r == '-' })
		if len(fields) != 4 {
			return fmt.Errorf("malformed nvidia installer: %q", s.NvidiaDriverVersion)
		}
		driverVersion = fields[3]
	} else {
		driverVersion = s.NvidiaDriverVersion
	}
	log.Println("Installing GPU drivers...")
	scriptPath := filepath.Join(s.NvidiaInstallDirHost, "setup_gpu.sh")
	if err := s.installScript(scriptPath, driverVersion); err != nil {
		return err
	}
	if err := s.runInstaller(scriptPath); err != nil {
		log.Println("Installing GPU drivers failed")
		return err
	}
	log.Println("Done installing GPU drivers")
	return nil
}
