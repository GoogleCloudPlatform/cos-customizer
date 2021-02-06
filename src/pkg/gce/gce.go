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

// Package gce contains high-level functionality for manipulating GCE resources.
package gce

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/config"

	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

const (
	defaultOperationTimeout = time.Duration(600) * time.Second
	defaultRetryInterval    = time.Duration(5) * time.Second
)

type timePkg struct {
	Now   func() time.Time
	Sleep func(time.Duration)
}

var (
	// ErrTimeout indicates that an operation timed out.
	ErrTimeout = errors.New("operation timed out")

	// ErrImageNotFound indicates that a GCE image could not be found
	ErrImageNotFound = errors.New("image not found")

	realTime = &timePkg{time.Now, time.Sleep}

	// This should match <prefix>-<channel>-<milestone>-<buildnumber>.
	// This is the format of images in cos-cloud.
	// Example: cos-dev-72-11172-0-0
	imageNameRegex = regexp.MustCompile("[a-z0-9-]+-[a-z]+-([0-9]+)-([0-9]+-[0-9]+-[0-9]+)")
)

// buildDeprecationStatus constructs a *compute.DeprecationStatus struct used in a Deprecate GCE API
// call. It fills in the structure with the "DEPRECATED" state, the given replacement, and the given
// delete time, if provided.
func buildDeprecationStatus(replacement string, deleteTime time.Time) *compute.DeprecationStatus {
	status := &compute.DeprecationStatus{State: "DEPRECATED", Replacement: replacement}
	if !deleteTime.IsZero() {
		status.Deleted = deleteTime.Format(time.RFC3339)
	}
	return status
}

func waitForOp(svc *compute.Service, project string, op *compute.Operation, deadline time.Time, t *timePkg) error {
	if op.Error != nil {
		return fmt.Errorf("error with operation. name: %s error: %v", op.Name, op.Error)
	}
	if op.Status == "DONE" {
		return nil
	}
	for {
		t.Sleep(defaultRetryInterval)
		op, err := svc.GlobalOperations.Get(project, op.Name).Do()
		if err != nil {
			return err
		}
		if op.Error != nil {
			return fmt.Errorf("error with operation. name: %s error: %v", op.Name, op.Error)
		}
		if op.Status == "DONE" {
			return nil
		}
		if t.Now().After(deadline) {
			return ErrTimeout
		}
	}
}

func waitForOps(svc *compute.Service, project string, ops []*compute.Operation, t *timePkg) error {
	deadline := t.Now().Add(defaultOperationTimeout)
	for _, op := range ops {
		if err := waitForOp(svc, project, op, deadline, t); err != nil {
			return err
		}
	}
	return nil
}

func deprecateInFamily(ctx context.Context, svc *compute.Service, newImage *config.Image, ttl int, t *timePkg) error {
	if newImage.Family == "" {
		return fmt.Errorf("input image does not have a family for deprecateInFamily. image: %v", newImage)
	}
	filter := fmt.Sprintf("(family = %s) (name != %s)", newImage.Family, newImage.Name)
	images := []*compute.Image{}
	err := svc.Images.List(newImage.Project).Filter(filter).Pages(ctx, func(imageList *compute.ImageList) error {
		images = append(images, imageList.Items...)
		return nil
	})
	if err != nil {
		return err
	}
	ops := []*compute.Operation{}
	for _, image := range images {
		if image.Deprecated != nil {
			continue
		}
		deleteTime := time.Time{}
		if ttl > 0 {
			deleteTime = t.Now().Add(time.Duration(ttl) * time.Second)
		}
		status := buildDeprecationStatus(newImage.URL(), deleteTime)
		op, err := svc.Images.Deprecate(newImage.Project, image.Name, status).Do()
		if err != nil {
			return err
		}
		ops = append(ops, op)
	}
	return waitForOps(svc, newImage.Project, ops, t)
}

// DeprecateInFamily deprecates all of the old images in an image family.
// Allows for assigning TTLs (in seconds) to deprecated images.
func DeprecateInFamily(ctx context.Context, svc *compute.Service, newImage *config.Image, ttl int) error {
	return deprecateInFamily(ctx, svc, newImage, ttl, realTime)
}

// ImageExists checks to see if the given image exists in the given project.
func ImageExists(svc *compute.Service, project, name string) (bool, error) {
	if _, err := svc.Images.Get(project, name).Do(); err != nil {
		if e, ok := err.(*googleapi.Error); ok && e.Code == http.StatusNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

type decodedImageName struct {
	name        string
	milestone   int
	buildNumber string
}

// newDecodedImageName decodes an image name from cos-cloud and returns
// image information encoded in that image name.
func newDecodedImageName(name string) (*decodedImageName, error) {
	match := imageNameRegex.FindStringSubmatch(name)
	if match == nil {
		return nil, fmt.Errorf("could not parse name %s", name)
	}
	milestone, err := strconv.Atoi(match[1])
	if err != nil {
		return nil, fmt.Errorf("could not convert %s to a milestone: %s", match[1], err)
	}
	return &decodedImageName{name, milestone, match[2]}, nil
}

func imageCompare(first, second *decodedImageName) bool {
	if first.milestone != second.milestone {
		return first.milestone < second.milestone
	}
	for i := 0; i < 3; i++ {
		// Because of how decodedImageNames are created (see newDecodedImageName),
		// these atoi operations are guaranteed to work.
		firstNum, _ := strconv.Atoi(strings.Split(first.buildNumber, "-")[i])
		secondNum, _ := strconv.Atoi(strings.Split(second.buildNumber, "-")[i])
		if firstNum != secondNum {
			return firstNum < secondNum
		}
	}
	return false
}

// ResolveMilestone gets the name of the latest COS image on the given milestone.
// This resolution is done by looking at the image names in cos-cloud.
func ResolveMilestone(ctx context.Context, svc *compute.Service, milestone int) (string, error) {
	var images []*compute.Image
	err := svc.Images.List("cos-cloud").Pages(ctx, func(imageList *compute.ImageList) error {
		images = append(images, imageList.Items...)
		return nil
	})
	if err != nil {
		return "", err
	}
	var inMilestone []*decodedImageName
	for _, image := range images {
		decoded, err := newDecodedImageName(image.Name)
		if err != nil {
			continue
		}
		if decoded.milestone == milestone {
			inMilestone = append(inMilestone, decoded)
		}
	}
	if len(inMilestone) == 0 {
		return "", ErrImageNotFound
	}
	sort.Slice(inMilestone, func(i, j int) bool {
		return imageCompare(inMilestone[i], inMilestone[j])
	})
	return inMilestone[len(inMilestone)-1].name, nil
}
