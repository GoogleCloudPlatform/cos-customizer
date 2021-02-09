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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/utils"
)

var (
	errStateAlreadyExists = errors.New("state already exists")
)

type stateData struct {
	Config      Config
	CurrentStep int
}

type state struct {
	dir  string
	data stateData
}

func (s *state) dataPath() string {
	return filepath.Join(s.dir, "state.json")
}

func (s *state) write() error {
	data, err := json.Marshal(&s.data)
	if err != nil {
		return fmt.Errorf("error marshalling JSON: %v", err)
	}
	if err := ioutil.WriteFile(s.dataPath(), data, 0660); err != nil {
		return fmt.Errorf("error writing %q: %v", s.dataPath(), err)
	}
	return nil
}

func downloadGCSObject(ctx context.Context, gcsClient *storage.Client, bucket, object, localPath string) error {
	address := fmt.Sprintf("gs://%s/%s", bucket, object)
	gcsObj, err := gcsClient.Bucket(bucket).Object(object).NewReader(ctx)
	if err != nil {
		return fmt.Errorf("error reading %q: %v", address, err)
	}
	defer utils.CheckClose(gcsObj, fmt.Sprintf("error closing GCS reader %q", address), &err)
	localFile, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer utils.CheckClose(localFile, "", &err)
	if _, err := io.Copy(localFile, gcsObj); err != nil {
		return fmt.Errorf("error copying %q to %q: %v", address, localFile.Name(), err)
	}
	return nil
}

func (s *state) unpackBuildContexts(ctx context.Context, deps Deps) (err error) {
	for name, address := range s.data.Config.BuildContexts {
		log.Printf("Unpacking build context %q from %q", name, address)
		if address[:len("gs://")] != "gs://" {
			return fmt.Errorf("cannot use address %q, only gs:// addresses are supported", address)
		}
		splitAddr := strings.SplitN(address[len("gs://"):], "/", 2)
		if len(splitAddr) != 2 || splitAddr[0] == "" || splitAddr[1] == "" {
			return fmt.Errorf("address %q is malformed", address)
		}
		bucket, object := splitAddr[0], splitAddr[1]
		tarPath := filepath.Join(s.dir, name+".tar")
		if err := downloadGCSObject(ctx, deps.GCSClient, bucket, object, tarPath); err != nil {
			return fmt.Errorf("error downloading %q to %q: %v", address, tarPath, err)
		}
		tarDir := filepath.Join(s.dir, name)
		if err := os.Mkdir(tarDir, 0770); err != nil {
			return err
		}
		args := []string{"xf", tarPath, "-C", tarDir}
		cmd := exec.Command(deps.TarCmd, args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf(`error in cmd "%s %v", see stderr for details: %v`, deps.TarCmd, args, err)
		}
		if err := os.Remove(tarPath); err != nil {
			return err
		}
	}
	return nil
}

func initState(ctx context.Context, deps Deps, dir string, c Config) (*state, error) {
	s := &state{dir: dir, data: stateData{Config: c, CurrentStep: 0}}
	if _, err := os.Stat(s.dataPath()); err == nil {
		return nil, errStateAlreadyExists
	}
	if err := os.MkdirAll(dir, 0770); err != nil {
		return nil, fmt.Errorf("error creating directory %q: %v", dir, err)
	}
	if err := s.write(); err != nil {
		return nil, err
	}
	if err := s.unpackBuildContexts(ctx, deps); err != nil {
		return nil, fmt.Errorf("error unpacking build contexts: %v", err)
	}
	return s, nil
}
