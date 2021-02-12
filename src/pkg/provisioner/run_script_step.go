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
	"log"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/utils"
)

type runScriptStep struct {
	BuildContext string
	Path         string
	Env          string
}

func (s *runScriptStep) run(runState *state) error {
	log.Printf("Executing script %q...", s.Path)
	buildContext := filepath.Join(runState.dir, s.BuildContext)
	script := filepath.Join(buildContext, s.Path)
	if err := utils.RunCommand([]string{"/bin/bash", script}, buildContext, strings.Split(s.Env, ",")); err != nil {
		return err
	}
	log.Printf("Done executing script %q", s.Path)
	return nil
}
