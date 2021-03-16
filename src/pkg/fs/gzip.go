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
	"compress/gzip"
	"fmt"
	"io"
	"os"

	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/utils"
)

// GzipFile compresses the file at the input path and saves the result at the
// output path.
func GzipFile(inPath, outPath string) (err error) {
	in, err := os.Open(inPath)
	if err != nil {
		return err
	}
	defer utils.CheckClose(in, fmt.Sprintf("error closing %q", inPath), &err)
	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer utils.CheckClose(out, fmt.Sprintf("error closing %q", outPath), &err)
	gzOut := gzip.NewWriter(out)
	defer utils.CheckClose(gzOut, fmt.Sprintf("error closing gzip writer for %q", outPath), &err)
	if _, err := io.Copy(gzOut, in); err != nil {
		return fmt.Errorf("error gzipping %q to %q: %v", inPath, outPath, err)
	}
	return nil
}
