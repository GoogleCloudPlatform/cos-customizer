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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
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

func initState(dir string, c Config) (*state, error) {
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
	return s, nil
}
