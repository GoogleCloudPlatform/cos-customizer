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
	"bytes"
	"context"
	"fmt"
	"testing"

	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/fakes"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestStore(t *testing.T) {
	var testData = []struct {
		testName string
		data     []byte
		object   string
	}{
		{"Empty", nil, "test-object"},
		{"NonEmpty", []byte("test-data"), "test-object"},
	}
	gcs := fakes.GCSForTest(t)
	defer gcs.Close()
	storageManager := gcsManager{gcs.Client, "bucket", "dir"}
	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			gcs.Objects = make(map[string][]byte)
			storageManager.store(context.Background(), bytes.NewReader(input.data), input.object)
			got, ok := gcs.Objects[fmt.Sprintf("/bucket/dir/cos-customizer/%s", input.object)]
			if !ok {
				t.Fatalf("gcsManager{}.store(_, %s, %s): could not find object", string(input.data), input.object)
			}
			if !cmp.Equal(got, input.data, cmpopts.EquateEmpty()) {
				t.Errorf("gcsManager{}.store(_, %s, %s) = %s, want %s", string(input.data), input.object, got, string(input.data))
			}

		})
	}
}

func TestManagedDirURL(t *testing.T) {
	storageManager := gcsManager{nil, "bucket", "dir"}
	if got := storageManager.managedDirURL(); got != "gs://bucket/dir/cos-customizer" {
		t.Errorf("gcsManager{}.managedDirURL() = %s, want gs://bucket/dir/cos-customizer", got)
	}
}

func TestURL(t *testing.T) {
	storageManager := gcsManager{nil, "bucket", "dir"}
	if got := storageManager.url("object"); got != "gs://bucket/dir/cos-customizer/object" {
		t.Errorf("gcsManager{}.url(object) = %s, want gs://bucket/dir/cos-customizer/object", got)
	}
}

func TestCleanup(t *testing.T) {
	ctx := context.Background()
	gcs := fakes.GCSForTest(t)
	defer gcs.Close()
	storageManager := gcsManager{gcs.Client, "bucket", "dir"}
	storageManager.store(ctx, bytes.NewReader(nil), "obj1")
	storageManager.store(ctx, bytes.NewReader(nil), "obj2")
	gcs.Objects["/bucket/obj3"] = nil
	storageManager.cleanup(ctx)
	if _, ok := gcs.Objects["/bucket/dir/cos-customizer/obj1"]; ok {
		t.Errorf("storageManager.cleanup(_): object /bucket/dir/cos-customizer/obj1 not deleted")
	}
	if _, ok := gcs.Objects["/bucket/dir/cos-customizer/obj2"]; ok {
		t.Errorf("storageManager.cleanup(_): object /bucket/dir/cos-customizer/obj2 not deleted")
	}
	if _, ok := gcs.Objects["/bucket/obj3"]; !ok {
		t.Errorf("storageManager.cleanup(_): object /bucket/obj3 was deleted")
	}
}
