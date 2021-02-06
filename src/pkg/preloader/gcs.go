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

package preloader

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
)

const (
	managedDir = "cos-customizer"
)

// gcsManager provides a simple key/value interface to a GCS directory, where keys are object paths
// and values are object data.
type gcsManager struct {
	gcsClient         *storage.Client
	gcsBucket, gcsDir string
}

func (m *gcsManager) managedDir() string {
	return filepath.Join(m.gcsDir, managedDir)
}

// managedDirURL gets the GCS URL of the directory being managed by the GCSManager.
func (m *gcsManager) managedDirURL() string {
	return fmt.Sprintf("gs://%s/%s", m.gcsBucket, m.managedDir())
}

func (m *gcsManager) objectPath(name string) string {
	return filepath.Join(m.managedDir(), name)
}

// store stores the given data in the given file. The file should be given as a path
// relative to the managed directory.
func (m *gcsManager) store(ctx context.Context, r io.Reader, name string) error {
	object := m.objectPath(name)
	w := m.gcsClient.Bucket(m.gcsBucket).Object(object).NewWriter(ctx)
	if _, err := io.Copy(w, r); err != nil {
		return err
	}
	return w.Close()
}

// url gets the GCS URL of the given file. The file should be given as a path
// relative to the managed directory.
func (m *gcsManager) url(name string) string {
	object := m.objectPath(name)
	return fmt.Sprintf("gs://%s/%s", m.gcsBucket, object)
}

// cleanup cleans up the managed directory.
func (m *gcsManager) cleanup(ctx context.Context) error {
	q := &storage.Query{Prefix: m.managedDir()}
	it := m.gcsClient.Bucket(m.gcsBucket).Objects(ctx, q)
	var objects []string
	for {
		objAttrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		objects = append(objects, objAttrs.Name)
	}
	for _, object := range objects {
		if err := m.gcsClient.Bucket(m.gcsBucket).Object(object).Delete(ctx); err != nil {
			return err
		}
	}
	return nil
}
