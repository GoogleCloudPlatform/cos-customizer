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

package config

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	compute "google.golang.org/api/compute/v1"
)

func TestSave(t *testing.T) {
	data := &struct{ Test string }{"test"}
	expected := "{\"Test\":\"test\"}"
	actual := new(strings.Builder)
	if err := Save(actual, data); err != nil {
		t.Fatal(err)
	}
	if got := actual.String(); got != expected {
		t.Errorf("actual: %s expected: %s", got, expected)
	}
}

func TestLoad(t *testing.T) {
	data := strings.NewReader("{\"Test\":\"test\"}")
	expected := &struct{ Test string }{"test"}
	actual := new(struct{ Test string })
	if err := Load(data, actual); err != nil {
		t.Fatal(err)
	}
	if *actual != *expected {
		t.Errorf("actual: %s expected: %s", actual, expected)
	}
}

func TestLoadFromFile(t *testing.T) {
	file := "testdata/test_1"
	expected := &Image{&compute.Image{Name: "test-name", Licenses: []string{}}, "test-project"}
	actual := new(Image)
	if err := LoadFromFile(file, actual); err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(actual, expected) {
		t.Errorf("actual: %v expected: %v", actual, expected)
	}
}

func TestImageMarshalJSON(t *testing.T) {
	image := NewImage("name", "project")
	bytes, err := json.Marshal(image)
	if err != nil {
		t.Fatal(err)
	}
	got := NewImage("", "")
	if err := json.Unmarshal(bytes, got); err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(got, image) {
		t.Errorf("actual: %v expected: %v", got, image)
	}
}
