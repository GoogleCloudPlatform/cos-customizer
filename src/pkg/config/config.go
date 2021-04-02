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

// Package config exports functionality for storing/retrieving build step configuration on/from
// the local disk.
package config

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/utils"
	compute "google.golang.org/api/compute/v1"
)

// Image stores GCE image configuration.
type Image struct {
	*compute.Image
	Project string
}

// NewImage builds an Image with some initialized defaults.
func NewImage(name string, project string) *Image {
	return &Image{
		&compute.Image{Name: name, Labels: make(map[string]string)},
		project,
	}
}

// URL gets the partial GCE URL of the image.
func (i *Image) URL() string {
	return fmt.Sprintf("projects/%s/global/images/%s", i.Project, i.Name)
}

// MarshalJSON marshals the image configuration into JSON.
func (i *Image) MarshalJSON() ([]byte, error) {
	computeImData, err := json.Marshal(i.Image)
	if err != nil {
		return nil, err
	}
	computeIm := make(map[string]interface{})
	if err := json.Unmarshal(computeImData, &computeIm); err != nil {
		return nil, err
	}
	computeIm["Project"] = i.Project
	return json.Marshal(computeIm)
}

// Build stores configuration data associated with the image build session.
type Build struct {
	GCSBucket string
	GCSDir    string
	Project   string
	Zone      string
	DiskSize  int
	GPUType   string
	Timeout   string
	GCSFiles  []string
}

// SaveConfigToFile clears the target config file and then saves the new config
// data.
func SaveConfigToFile(configFile *os.File, v interface{}) error {
	if _, err := configFile.Seek(0, 0); err != nil {
		return fmt.Errorf("cannot seek config file, error msg:(%v)", err)
	}
	if err := configFile.Truncate(0); err != nil {
		return fmt.Errorf("cannot truncate config file, error msg:(%v)", err)
	}
	if err := Save(configFile, v); err != nil {
		return fmt.Errorf("cannot save config file, error msg:(%v)", err)
	}
	return nil
}

// SaveConfigToPath does the same thing as SaveConfigToFile, but updates a file
// system path.
func SaveConfigToPath(configPath string, config interface{}) (err error) {
	configFile, err := os.OpenFile(configPath, os.O_RDWR, 0666)
	if err != nil {
		return err
	}
	defer utils.CheckClose(configFile, "error writing to "+configPath, &err)
	return SaveConfigToFile(configFile, config)
}

// Save serializes the given struct as JSON and writes it out.
func Save(w io.Writer, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, string(data))
	return err
}

// Load deserializes JSON formatted data into the given struct.
func Load(r io.Reader, v interface{}) error {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

// LoadFromFile loads JSON data from a file into the given struct.
func LoadFromFile(path string, v interface{}) (err error) {
	r, err := os.Open(path)
	if err != nil {
		return err
	}
	defer utils.CheckClose(r, fmt.Sprintf("error closing %q", path), &err)
	return Load(r, v)
}
