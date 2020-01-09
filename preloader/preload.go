// Copyright 2018 Google LLC
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

// Package preloader contains functionality for preloading a COS image from
// provided configuration.
package preloader

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"cos-customizer/config"
	"cos-customizer/fs"

	"cloud.google.com/go/storage"
	yaml "gopkg.in/yaml.v2"
)

func buildCloudConfig(script io.Reader, service io.Reader) (string, error) {
	scriptContent, err := ioutil.ReadAll(script)
	if err != nil {
		return "", err
	}
	serviceContent, err := ioutil.ReadAll(service)
	if err != nil {
		return "", err
	}
	cloudConfig := make(map[string]interface{})
	scriptEntry := map[string]string{
		"path":        "/tmp/startup.sh",
		"permissions": "0644",
		"content":     string(scriptContent),
	}
	serviceEntry := map[string]string{
		"path":        "/etc/systemd/system/customizer.service",
		"permissions": "0644",
		"content":     string(serviceContent),
	}
	cloudConfig["write_files"] = []map[string]string{
		scriptEntry,
		serviceEntry,
	}
	cloudConfig["runcmd"] = []string{
		"echo \"Starting startup service...\"",
		"systemctl daemon-reload",
		"systemctl --no-block start customizer.service",
	}
	cloudConfigYaml, err := yaml.Marshal(&cloudConfig)
	if err != nil {
		return "", err
	}
	return "#cloud-config\n\n" + string(cloudConfigYaml), nil
}

// writeCloudConfig composes a cloud-config from the given script and systemd service and writes the result
// to a temporary file.
func writeCloudConfig(scriptPath string, servicePath string) (string, error) {
	scriptReader, err := os.Open(scriptPath)
	if err != nil {
		return "", err
	}
	defer scriptReader.Close()
	serviceReader, err := os.Open(servicePath)
	if err != nil {
		return "", err
	}
	defer serviceReader.Close()
	cloudConfig, err := buildCloudConfig(scriptReader, serviceReader)
	if err != nil {
		return "", err
	}
	w, err := ioutil.TempFile(fs.ScratchDir, "cloudconfig-")
	if err != nil {
		return "", err
	}
	if _, err := w.WriteString(cloudConfig); err != nil {
		w.Close()
		os.Remove(w.Name())
		return "", err
	}
	if err := w.Close(); err != nil {
		os.Remove(w.Name())
		return "", err
	}
	return w.Name(), nil
}

// storeInGCS stores the given files in GCS using the given gcsManager.
// Input files are stored in GCS using their basename. Input files should not have the same
// basename to avoid collisions.
func storeInGCS(ctx context.Context, gcs *gcsManager, files []string) error {
	basenames := make(map[string]bool)
	for _, file := range files {
		if _, ok := basenames[filepath.Base(file)]; ok {
			return fmt.Errorf("storeInGCS: collision in basename %s", filepath.Base(file))
		}
		basenames[filepath.Base(file)] = true
	}
	for _, file := range files {
		r, err := os.Open(file)
		if err != nil {
			return err
		}
		defer r.Close()
		if err := gcs.store(ctx, r, filepath.Base(file)); err != nil {
			return err
		}
	}
	return nil
}

// writeDaisyWorkflow templates the given Daisy workflow and writes the result to a temporary file.
// The given workflow should be the one at //data/build_image.wf.json.
func writeDaisyWorkflow(inputWorkflow string, outputImage *config.Image, buildSpec *config.Build) (string, error) {
	tmplContents, err := ioutil.ReadFile(inputWorkflow)
	if err != nil {
		return "", err
	}
	labelsJSON, err := json.Marshal(outputImage.Labels)
	if err != nil {
		return "", err
	}
	acceleratorsJSON, err := json.Marshal([]map[string]interface{}{})
	if buildSpec.GPUType != "" {
		acceleratorType := fmt.Sprintf("projects/%s/zones/%s/acceleratorTypes/%s",
			buildSpec.Project, buildSpec.Zone, buildSpec.GPUType)
		acceleratorsJSON, err = json.Marshal([]map[string]interface{}{
			{"acceleratorType": acceleratorType, "acceleratorCount": 1}})
	}
	if err != nil {
		return "", err
	}
	licensesJSON, err := json.Marshal(outputImage.Licenses)
	if err != nil {
		return "", err
	}
	tmpl, err := template.New("workflow").Parse(string(tmplContents))
	if err != nil {
		return "", err
	}
	w, err := ioutil.TempFile(fs.ScratchDir, "daisy-")
	if err != nil {
		return "", err
	}
	if err := tmpl.Execute(w, struct {
		Labels       string
		Accelerators string
		Licenses     string
	}{
		string(labelsJSON),
		string(acceleratorsJSON),
		string(licensesJSON),
	}); err != nil {
		w.Close()
		os.Remove(w.Name())
		return "", err
	}
	if err := w.Close(); err != nil {
		os.Remove(w.Name())
		return "", err
	}
	return w.Name(), nil
}

func sanitize(output *config.Image) {
	var licenses []string
	for _, l := range output.Licenses {
		if l != "" {
			licenses = append(licenses, strings.TrimPrefix(l, "https://www.googleapis.com/compute/v1/"))
		}
	}
	output.Licenses = licenses
}

// daisyArgs computes the parameters to the cos-customizer Daisy workflow (//data/build_image.wf.json)
// and uploads dependencies to GCS.
func daisyArgs(ctx context.Context, gcs *gcsManager, files *fs.Files, input *config.Image, output *config.Image, buildSpec *config.Build) ([]string, error) {
	sanitize(output)
	toUpload := []string{
		files.UserBuildContextArchive,
		files.BuiltinBuildContextArchive,
		files.StateFile,
	}
	if err := storeInGCS(ctx, gcs, toUpload); err != nil {
		return nil, err
	}
	daisyWorkflow, err := writeDaisyWorkflow(files.DaisyWorkflow, output, buildSpec)
	if err != nil {
		return nil, err
	}
	cloudConfigFile, err := writeCloudConfig(files.StartupScript, files.SystemdService)
	if err != nil {
		return nil, err
	}
	var args []string
	if buildSpec.DiskSize != 0 {
		args = append(args, "-var:disk_size_gb", strconv.Itoa(buildSpec.DiskSize))
	}
	if output.Family != "" {
		args = append(args, "-var:output_image_family", output.Family)
	}
	hostMaintenance := "MIGRATE"
	if buildSpec.GPUType != "" {
		hostMaintenance = "TERMINATE"
	}
	args = append(
		args,
		"-var:source_image",
		input.URL(),
		"-var:output_image_name",
		output.Name,
		"-var:output_image_project",
		output.Project,
		"-var:user_build_context",
		gcs.url(filepath.Base(files.UserBuildContextArchive)),
		"-var:builtin_build_context",
		gcs.url(filepath.Base(files.BuiltinBuildContextArchive)),
		"-var:state_file",
		gcs.url(filepath.Base(files.StateFile)),
		"-var:cloud_config",
		cloudConfigFile,
		"-var:host_maintenance",
		hostMaintenance,
		"-gcs_path",
		gcs.managedDirURL(),
		"-project",
		buildSpec.Project,
		"-zone",
		buildSpec.Zone,
		"-default_timeout",
		buildSpec.Timeout,
		daisyWorkflow,
	)
	return args, nil
}

// BuildImage builds a customized image using Daisy.
func BuildImage(ctx context.Context, gcsClient *storage.Client, files *fs.Files, input, output *config.Image,
	buildSpec *config.Build) error {
	gcs := &gcsManager{gcsClient, buildSpec.GCSBucket, buildSpec.GCSDir}
	defer gcs.cleanup(ctx)
	args, err := daisyArgs(ctx, gcs, files, input, output, buildSpec)
	if err != nil {
		return err
	}
	cmd := exec.Command(files.DaisyBin, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stdout
	return cmd.Run()
}
