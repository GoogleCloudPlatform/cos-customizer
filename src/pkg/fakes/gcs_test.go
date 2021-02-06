// Copyright 2019 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fakes

import (
	"context"
	"fmt"
	"io/ioutil"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/api/iterator"
)

func TestReadObject(t *testing.T) {
	gcs := GCSForTest(t)
	defer gcs.Close()
	gcs.Objects["/test-bucket/test-object"] = []byte("data")
	r, err := gcs.Client.Bucket("test-bucket").Object("test-object").NewReader(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	got, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(got, []byte("data")) {
		t.Errorf("bucket 'test-bucket', object 'test-object' has %s; want 'data'", string(got))
	}
}

func TestReadObjectNotFound(t *testing.T) {
	gcs := GCSForTest(t)
	defer gcs.Close()
	_, err := gcs.Client.Bucket("test-bucket").Object("test-object").NewReader(context.Background())
	if err != storage.ErrObjectNotExist {
		t.Errorf("bucket 'test-bucket', object 'test-object' has %s; want %s", err, storage.ErrObjectNotExist)
	}
}

func TestIterate(t *testing.T) {
	testData := []struct {
		testName        string
		prefix          string
		objects         map[string][]byte
		bucket          string
		expectedObjects []string
	}{
		{
			"HasItems",
			"",
			map[string][]byte{
				"/test-bucket/obj-1": []byte(""),
				"/test-bucket/obj-2": []byte(""),
				"/test-bucket/obj-3": []byte(""),
				"/bucket/obj-4":      []byte(""),
			},
			"test-bucket",
			[]string{
				"/test-bucket/obj-1",
				"/test-bucket/obj-2",
				"/test-bucket/obj-3",
			},
		},
		{
			"NoItems",
			"",
			make(map[string][]byte),
			"test-bucket",
			nil,
		},
		{
			"HasPrefix",
			"pre",
			map[string][]byte{
				"/test-bucket/pre-1": []byte(""),
				"/test-bucket/pre-2": []byte(""),
				"/test-bucket/obj-3": []byte(""),
			},
			"test-bucket",
			[]string{
				"/test-bucket/pre-1",
				"/test-bucket/pre-2",
			},
		},
	}
	gcs := GCSForTest(t)
	defer gcs.Close()
	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			gcs.Objects = input.objects
			q := &storage.Query{
				Delimiter: "",
				Prefix:    input.prefix,
				Versions:  false,
			}
			it := gcs.Client.Bucket(input.bucket).Objects(context.Background(), q)
			var actualObjects []string
			for {
				objAttrs, err := it.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				actualObjects = append(actualObjects, fmt.Sprintf("/%s/%s", objAttrs.Bucket, objAttrs.Name))
			}
			sortStrSlices := cmpopts.SortSlices(func(a, b string) bool { return a < b })
			if !cmp.Equal(actualObjects, input.expectedObjects, sortStrSlices) {
				t.Errorf("bucket %s has %v, want %v", input.bucket, actualObjects, input.expectedObjects)
			}
		})
	}
}

func TestWrite(t *testing.T) {
	testData := []struct {
		testName string
		object   string
		bucket   string
		data     []byte
	}{
		{
			"NonEmptyWrite",
			"test-object",
			"test-bucket",
			[]byte("data"),
		},
		{
			"EmptyWrite",
			"test-object",
			"test-bucket",
			nil,
		},
	}
	gcs := GCSForTest(t)
	defer gcs.Close()
	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			gcs.Objects = make(map[string][]byte)
			w := gcs.Client.Bucket(input.bucket).Object(input.object).NewWriter(context.Background())
			w.Write(input.data)
			w.Close()
			if got, ok := gcs.Objects[fmt.Sprintf("/%s/%s", input.bucket, input.object)]; !ok {
				t.Errorf("bucket %s, object %s does not exist", input.bucket, input.object)
			} else if !cmp.Equal(got, input.data, cmpopts.EquateEmpty()) {
				t.Errorf("bucket %s, object %s has %v; want %v", input.bucket, input.object, got, input.data)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	testData := []struct {
		testName string
		objects  []string
		toDelete []string
		want     []string
	}{
		{
			"OneObject",
			[]string{"/bucket/obj1"},
			[]string{"obj1"},
			nil,
		},
		{
			"HasLeftovers",
			[]string{"/bucket/obj1", "/bucket/obj2"},
			[]string{"obj1"},
			[]string{"/bucket/obj2"},
		},
		{
			"MultipleObjects",
			[]string{"/bucket/obj1", "/bucket/obj2", "/bucket/obj3"},
			[]string{"obj1", "obj2"},
			[]string{"/bucket/obj3"},
		},
	}
	gcs := GCSForTest(t)
	defer gcs.Close()
	for _, input := range testData {
		t.Run(input.testName, func(t *testing.T) {
			for _, k := range input.objects {
				gcs.Objects[k] = []byte("data")
			}
			for _, object := range input.toDelete {
				if err := gcs.Client.Bucket("bucket").Object(object).Delete(context.Background()); err != nil {
					t.Fatalf("Delete() failed - got err: %v, want: nil", err)
				}
			}
			var got []string
			it := gcs.Client.Bucket("bucket").Objects(context.Background(), &storage.Query{})
			for {
				objAttrs, err := it.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					t.Fatalf("Iterate failed - got err: %v, want: nil", err)
				}
				got = append(got, fmt.Sprintf("/%s/%s", objAttrs.Bucket, objAttrs.Name))
			}
			sortStrSlices := cmpopts.SortSlices(func(a, b string) bool { return a < b })
			if !cmp.Equal(got, input.want, cmpopts.EquateEmpty(), sortStrSlices) {
				t.Errorf("Tried to delete %v: got %v, want %v", input.toDelete, got, input.want)
			}
		})
	}
}

func TestDeleteObjectDoesNotExist(t *testing.T) {
	gcs := GCSForTest(t)
	defer gcs.Close()
	err := gcs.Client.Bucket("bucket").Object("object").Delete(context.Background())
	if err != storage.ErrObjectNotExist {
		t.Logf("objects: %v", gcs.Objects)
		t.Errorf("delete object 'object' in bucket 'bucket' has %s; want %s", err, storage.ErrObjectNotExist)
	}

}
