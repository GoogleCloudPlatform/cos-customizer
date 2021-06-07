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
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"text/template"

	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/utils"
)

type InstallPackagesStep struct {
	BuildContext                 string
	PkgSpecURL                   string
	AnthosInstallerDir           string
	AnthosInstallerVersion       string
	AnthosInstallerReleaseBucket string
}

// setDefaultAnthosInstallerDir sets the AnthosInstallerDir to the input dir path.
func (ip *InstallPackagesStep) setDefaultAnthosInstallerDir(dir string) {
	// AnthosInstallerDir is the place where the anthos_installer is
	// installed.
	if ip.AnthosInstallerDir == "" {
		ip.AnthosInstallerDir = dir
	}
}

// runInstaller runs the anthos-installer installing the packages mentioned in the pkg spec.
func (ip *InstallPackagesStep) runInstaller(buildContext string) (err error) {
	scriptPath := filepath.Join(ip.AnthosInstallerDir, "anthos_installer_install.sh")
	f, err := os.OpenFile(scriptPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0744)
	if err != nil {
		return err
	}
	defer utils.CheckClose(f, fmt.Sprintf("error closing %s", scriptPath), &err)
	t, err := template.New("anthos-installer-install-script").Parse(anthosInstallerInstallTemplateScript)
	if err != nil {
		return err
	}
	// if the given input PkgSpecURL is a valid URL then it i
	pkgSpecURL := ip.PkgSpecURL
	if !checkIfRemoteURL(ip.PkgSpecURL) {
		pkgSpecURL = filepath.Join(buildContext, ip.PkgSpecURL)
	}
	if err := t.Execute(f, &InstallPackagesStep{
		PkgSpecURL:             utils.QuoteForShell(pkgSpecURL),
		AnthosInstallerDir:     utils.QuoteForShell(ip.AnthosInstallerDir),
		AnthosInstallerVersion: utils.QuoteForShell(ip.AnthosInstallerVersion),
	}); err != nil {
		return fmt.Errorf("error installing %q: %v", scriptPath, err)
	}
	return utils.RunCommand([]string{"/bin/bash", scriptPath}, "", nil)
}

func (ip *InstallPackagesStep) run(ctx context.Context, runState *state, deps *stepDeps) error {
	log.Printf("Installing Packages from the %s...", ip.PkgSpecURL)
	buildContext := filepath.Join(runState.dir, ip.BuildContext)
	ip.setDefaultAnthosInstallerDir(runState.dir)
	//set up the installer at the AnthosInstallerDir
	anthosInstallerTar := filepath.Join(ip.AnthosInstallerDir, ip.AnthosInstallerVersion)
	// Downloading the anthos installer file from the storage bucket.
	if err := downloadGCSObject(ctx, deps.GCSClient, ip.AnthosInstallerReleaseBucket, ip.AnthosInstallerVersion, anthosInstallerTar); err != nil {
		return err
	}
	if err := ip.runInstaller(buildContext); err != nil {
		return err
	}
	log.Printf("Done Installing the Packages from %s", ip.PkgSpecURL)
	return nil
}

// checkIfRemoteURL returns "true" if filePath is a remote file
func checkIfRemoteURL(filePath string) bool {
	p, err := url.Parse(filePath)
	if err != nil {
		return false
	}
	return p.Scheme == "http" || p.Scheme == "https" || p.Scheme == "gs"
}
