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

package fs

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestGzipFile(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "test-gzip-file-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	data, err := os.Create(filepath.Join(tmpDir, "data"))
	if err != nil {
		t.Fatal(err)
	}
	defer data.Close()
	if _, err := io.Copy(data, bytes.NewReader([]byte("aaaaa"))); err != nil {
		t.Fatal(err)
	}
	if err := data.Sync(); err != nil {
		t.Fatal(err)
	}
	outPath := filepath.Join(tmpDir, "out")
	signature := fmt.Sprintf("GzipFile(%q, %q)", data.Name(), outPath)
	if err := GzipFile(data.Name(), outPath); err != nil {
		t.Fatalf("%s = %v; want nil", signature, err)
	}
	outFile, err := os.Open(outPath)
	if err != nil {
		t.Fatal(err)
	}
	defer outFile.Close()
	uncompressed, err := gzip.NewReader(outFile)
	if err != nil {
		t.Fatal(err)
	}
	defer uncompressed.Close()
	gotBytes, err := ioutil.ReadAll(uncompressed)
	if err != nil {
		t.Fatal(err)
	}
	got := string(gotBytes)
	if want := "aaaaa"; got != want {
		t.Errorf("%s = %q; want %q", signature, got, want)
	}
}
