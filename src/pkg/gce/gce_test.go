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

package gce

import (
	"context"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/config"
	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/fakes"

	compute "google.golang.org/api/compute/v1"
)

func fakeTime(current time.Time) *timePkg {
	fake := fakes.NewTime(current)
	return &timePkg{fake.Now, fake.Sleep}
}

func TestDeprecateInFamilyNoFamily(t *testing.T) {
	ctx := context.Background()
	newImage := &config.Image{&compute.Image{Name: "test-name"}, "test-project"}
	if err := DeprecateInFamily(ctx, nil, newImage, 0); err == nil {
		t.Error("DeprecateInFamily: did not fail when input image had no family")
	}
}

func TestDeprecateInFamilyNoItems(t *testing.T) {
	fakeGCE, client := fakes.GCEForTest(t, "test-project")
	defer fakeGCE.Close()
	ctx := context.Background()
	newImage := &config.Image{&compute.Image{Name: "test-name", Family: "test-family"}, "test-project"}
	if err := DeprecateInFamily(ctx, client, newImage, 0); err != nil {
		t.Logf("DeprecateInFamily(_, _, %v, 0)", newImage)
		t.Fatal(err)
	}
	if len(fakeGCE.Deprecated) != 0 {
		t.Errorf("an image was deprecated. Map: %v", fakeGCE.Deprecated)
	}
}

func TestDeprecateInFamilyIgnoreDeprecated(t *testing.T) {
	fakeGCE, client := fakes.GCEForTest(t, "test-project")
	defer fakeGCE.Close()
	ctx := context.Background()
	fakeGCE.Images.Items = []*compute.Image{{Deprecated: &compute.DeprecationStatus{}}}
	newImage := &config.Image{&compute.Image{Name: "test-name", Family: "test-family"}, "test-project"}
	if err := DeprecateInFamily(ctx, client, newImage, 0); err != nil {
		t.Logf("fakeGCE.Images.Items: %v", fakeGCE.Images.Items)
		t.Logf("DeprecateInFamily(_, _, %v, 0)", newImage)
		t.Fatal(err)
	}
	if len(fakeGCE.Deprecated) != 0 {
		t.Errorf("an image was deprecated. Map: %v", fakeGCE.Deprecated)
	}
}

func TestDeprecateInFamily(t *testing.T) {
	testDeprecateInFamilyData := []struct {
		testName string
		images   []*compute.Image
		ops      []*compute.Operation
	}{
		{
			"DoneInstantly",
			[]*compute.Image{
				{Name: "dep-1"},
			},
			[]*compute.Operation{
				{Name: "op-1", Status: "DONE"},
			},
		},
		{
			"RunningOpBeforeDone",
			[]*compute.Image{
				{Name: "dep-1"},
			},
			[]*compute.Operation{
				{Name: "op-1", Status: "RUNNING"},
				{Name: "op-2", Status: "DONE"},
			},
		},
		{
			"PendingOpBeforeDone",
			[]*compute.Image{
				{Name: "dep-1"},
			},
			[]*compute.Operation{
				{Name: "op-1", Status: "PENDING"},
				{Name: "op-2", Status: "DONE"},
			},
		},
		{
			"TwoRunningBeforeDone",
			[]*compute.Image{
				{Name: "dep-1"},
			},
			[]*compute.Operation{
				{Name: "op-1", Status: "RUNNING"},
				{Name: "op-2", Status: "RUNNING"},
				{Name: "op-3", Status: "DONE"},
			},
		},
		{
			"TwoImages",
			[]*compute.Image{
				{Name: "dep-1"},
				{Name: "dep-2"},
			},
			[]*compute.Operation{
				{Name: "op-1", Status: "DONE"},
				{Name: "op-2", Status: "DONE"},
			},
		},
		{
			"TwoImagesRunningBeforeDone",
			[]*compute.Image{
				{Name: "dep-1"},
				{Name: "dep-2"},
			},
			[]*compute.Operation{
				{Name: "op-1", Status: "RUNNING"},
				{Name: "op-1", Status: "DONE"},
				{Name: "op-2", Status: "RUNNING"},
				{Name: "op-2", Status: "DONE"},
			},
		},
	}
	fakeGCE, client := fakes.GCEForTest(t, "test-project")
	defer fakeGCE.Close()
	for _, input := range testDeprecateInFamilyData {
		t.Run(input.testName, func(t *testing.T) {
			date := time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)
			ctx := context.Background()
			fakeGCE.Images.Items = input.images
			fakeGCE.Operations = input.ops
			newImage := &config.Image{&compute.Image{Name: "test-name", Family: "test-family"}, "test-project"}
			if err := deprecateInFamily(ctx, client, newImage, 0, fakeTime(date)); err != nil {
				t.Logf("input: %v", input)
				t.Logf("deprecateInFamily(_, _, %v, 0, _)", newImage)
				t.Fatal(err)
			}
			if len(fakeGCE.Deprecated) != len(input.images) {
				t.Fatalf("deprecated: %v actual: %d expected: %d", fakeGCE.Deprecated, len(fakeGCE.Deprecated),
					len(input.images))
			}
			for _, image := range input.images {
				status := fakeGCE.Deprecated[image.Name]
				if status.State != "DEPRECATED" {
					t.Errorf("image: %v actual: %s expected: DEPRECATED", image, status.State)
				}
				if want := newImage.URL(); status.Replacement != want {
					t.Errorf("image: %v actual: %s expected: %s", image, status.Replacement, want)
				}
			}
		})
	}
}

func TestDeprecateInFamilyTimeout(t *testing.T) {
	date := time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)
	fakeGCE, client := fakes.GCEForTest(t, "test-project")
	defer fakeGCE.Close()
	ctx := context.Background()
	fakeGCE.Images.Items = []*compute.Image{{Name: "dep-1"}}
	fakeGCE.Operations = nil
	for i := 0; i < 1000; i++ {
		fakeGCE.Operations = append(fakeGCE.Operations, &compute.Operation{Name: "", Status: "RUNNING"})
	}
	newImage := &config.Image{&compute.Image{Name: "test-name", Family: "test-family"}, "test-project"}
	if err := deprecateInFamily(ctx, client, newImage, 0, fakeTime(date)); err != ErrTimeout {
		t.Errorf("operation did not timeout. err: %s", err)
	}
}

func TestDeprecateInFamilyTTL(t *testing.T) {
	date := time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)
	fakeGCE, client := fakes.GCEForTest(t, "test-project")
	defer fakeGCE.Close()
	ctx := context.Background()
	fakeGCE.Images.Items = []*compute.Image{{Name: "dep-1"}}
	fakeGCE.Operations = []*compute.Operation{{Name: "op-1", Status: "DONE"}}
	newImage := &config.Image{&compute.Image{Name: "test-name", Family: "test-family"}, "test-project"}
	if err := deprecateInFamily(ctx, client, newImage, 30, fakeTime(date)); err != nil {
		t.Logf("fakeGCE.Images.Items: %v", fakeGCE.Images.Items)
		t.Logf("fakeGCE.Operations: %v", fakeGCE.Operations)
		t.Logf("deprecateInFamily(_, _, %v, 30, _)", newImage)
		t.Fatal(err)
	}
	status := fakeGCE.Deprecated["dep-1"]
	expected := date.Add(time.Duration(30) * time.Second).Format(time.RFC3339)
	if status.Deleted != expected {
		t.Errorf("actual: %s expected: %s", status.Deleted, expected)
	}
}

func TestImageExists(t *testing.T) {
	testImageExistsData := []struct {
		testName string
		images   []*compute.Image
		name     string
		expected bool
	}{
		{
			"DoesntExist",
			[]*compute.Image{{Name: "im-1"}},
			"im-2",
			false,
		},
		{
			"Exists",
			[]*compute.Image{{Name: "im-1"}},
			"im-1",
			true,
		},
	}
	fakeGCE, client := fakes.GCEForTest(t, "test-project")
	defer fakeGCE.Close()
	for _, input := range testImageExistsData {
		t.Run(input.testName, func(t *testing.T) {
			fakeGCE.Images.Items = input.images
			actual, err := ImageExists(client, "test-project", input.name)
			if err != nil {
				t.Logf("ImageExists(_, _, %s): %t", input.name, actual)
				t.Logf("fakeGCE.Images.Items: %v", fakeGCE.Images.Items)
				t.Fatal(err)
			}
			if actual != input.expected {
				t.Logf("ImageExists(_, _, %s): %t", input.name, actual)
				t.Logf("fakeGCE.Images.Items: %v", fakeGCE.Images.Items)
				t.Errorf("actual: %v expected: %v", actual, input.expected)
			}
		})
	}
}

func buildImageList(names []string) []*compute.Image {
	var images []*compute.Image
	for _, name := range names {
		images = append(images, &compute.Image{Name: name})
	}
	return images
}

func TestResolveMilestone(t *testing.T) {
	testResolveMilestoneData := []struct {
		testName      string
		names         []string
		milestone     int
		expected      string
		expectedError error
	}{
		{
			"OneCandidate",
			[]string{"cos-dev-68-10718-0-0", "cos-beta-67-10525-0-0"},
			68,
			"cos-dev-68-10718-0-0",
			nil,
		},
		{
			"TwoCandidates",
			[]string{"cos-dev-68-10718-11-0", "cos-dev-68-10718-0-0", "bad-image"},
			68,
			"cos-dev-68-10718-11-0",
			nil,
		},
		{
			"NoImages",
			nil,
			68,
			"",
			ErrImageNotFound,
		},
		{
			"NoCandidates",
			[]string{"bad-image"},
			68,
			"",
			ErrImageNotFound,
		},
	}
	fakeGCE, client := fakes.GCEForTest(t, "cos-cloud")
	defer fakeGCE.Close()
	for _, input := range testResolveMilestoneData {
		t.Run(input.testName, func(t *testing.T) {
			ctx := context.Background()
			fakeGCE.Images.Items = buildImageList(input.names)
			actual, err := ResolveMilestone(ctx, client, input.milestone)
			if err != input.expectedError {
				t.Errorf("ResolveMilestone(_, _, %v) = %s, want: %s", input.milestone, err, input.expectedError)
			}
			if actual != input.expected {
				t.Errorf("ResolveMilestone(_, _, %v) = %s, want: %s", input.milestone, actual, input.expected)
			}
		})
	}
}
