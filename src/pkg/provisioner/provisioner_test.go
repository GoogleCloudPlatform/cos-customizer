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
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestStateExists(t *testing.T) {
	dir, err := ioutil.TempDir("", "provisioner-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	if err := ioutil.WriteFile(filepath.Join(dir, "state.json"), []byte("{}"), 0660); err != nil {
		t.Fatal(err)
	}
	config := Config{}
	if err := Run(dir, config); err != errStateAlreadyExists {
		t.Fatalf("Run(%s, %+v) = %v; want %v", dir, config, err, errStateAlreadyExists)
	}
}
