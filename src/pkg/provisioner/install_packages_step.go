package provisioner

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"text/template"

	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/utils"
)

type InstallPackagesStep struct {
	BuildContext           string
	PkgSpecDir             string
	TopWorkDir             string
	AnthosInstallerVersion string
	AnthosInstallerDir     string
}

func (ip *InstallPackagesStep) setDefaults() {
	if ip.AnthosInstallerDir == "" {
		ip.AnthosInstallerDir = "/home/anthos_installer"
	}
}

func (ip *InstallPackagesStep) runInstaller(buildContext string) error {
	scriptPath := filepath.Join(ip.AnthosInstallerDir, "anthos_installer_install.sh")
	if err := os.MkdirAll(filepath.Dir(scriptPath), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(scriptPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0744)
	if err != nil {
		return err
	}
	defer utils.CheckClose(f, fmt.Sprintf("error closing %q", scriptPath), &err)
	t, err := template.New("anthos-installer-install-script").Parse(anthosInstallerInstallTemplateScript)
	if err != nil {
		return err
	}

	pkgSpecDir := filepath.Join(buildContext, ip.PkgSpecDir)
	if err := t.Execute(f, &InstallPackagesStep{
		PkgSpecDir:             utils.QuoteForShell(pkgSpecDir),
		AnthosInstallerVersion: utils.QuoteForShell(ip.AnthosInstallerVersion),
		TopWorkDir:             utils.QuoteForShell(ip.TopWorkDir),
		AnthosInstallerDir:     utils.QuoteForShell(ip.AnthosInstallerDir),
	}); err != nil {
		return fmt.Errorf("error installing %q: %v", scriptPath, err)
	}
	if err := utils.RunCommand([]string{"/bin/bash", scriptPath}, "", nil); err != nil {
		return err
	}
	return nil
}

func (ip *InstallPackagesStep) run(runState *state) error {
	log.Printf("Installing Packages from the %q...", ip.PkgSpecDir)
	ip.setDefaults()
	buildContext := filepath.Join(runState.dir, ip.BuildContext)
	if err := ip.runInstaller(buildContext); err != nil {
		return err
	}
	log.Printf("Done Installing the Packages from %q", ip.PkgSpecDir)
	return nil
}
