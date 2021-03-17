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

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"time"

	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/config"
	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/fs"
	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/gce"
	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/preloader"
	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/provisioner"
	"github.com/GoogleCloudPlatform/cos-customizer/src/pkg/tools/partutil"

	"github.com/google/subcommands"
)

// FinishImageBuild implements subcommands.Command for the "finish-image-build" command.
// This command finishes an image build by converting saved image configurations into
// an actual GCE image.
type FinishImageBuild struct {
	imageProject   string
	zone           string
	project        string
	imageName      string
	imageSuffix    string
	imageFamily    string
	deprecateOld   bool
	oldImageTTLSec int
	labels         *mapVar
	licenses       *listVar
	inheritLabels  bool
	oemSize        string
	oemFSSize4K    uint64
	diskSize       int
	timeout        time.Duration
}

// Name implements subcommands.Command.Name.
func (f *FinishImageBuild) Name() string {
	return "finish-image-build"
}

// Synopsis implements subcommands.Command.Synopsis.
func (f *FinishImageBuild) Synopsis() string {
	return "Complete the COS image build and generate a GCE image."
}

// Usage implements subcommands.Command.Usage.
func (f *FinishImageBuild) Usage() string {
	return `finish-image-build [flags]
`
}

// SetFlags implements subcommands.Command.SetFlags.
func (f *FinishImageBuild) SetFlags(flags *flag.FlagSet) {
	flags.StringVar(&f.imageProject, "image-project", "", "Output image project.")
	flags.StringVar(&f.imageName, "image-name", "", "Output image name. Mutually exclusive with 'image-suffix'.")
	flags.StringVar(&f.imageSuffix, "image-suffix", "", "Construct the output image name from the input image "+
		"name and this suffix. Mutually exclusive with 'image-name'.")
	flags.StringVar(&f.imageFamily, "image-family", "", "Output image family.")
	flags.BoolVar(&f.deprecateOld, "deprecate-old-images", false, "Deprecate old images in the output image "+
		"family. Can only be used if 'image-family' is set.")
	flags.IntVar(&f.oldImageTTLSec, "old-image-ttl", 0, "Time-to-live in seconds for old images that are "+
		"deprecated. After this period of time, old images will enter the deleted state. Can only be used if "+
		"'deprecate-old-images' is set. '0' indicates no time-to-live (images won't be configured to enter "+
		"the deleted state).")
	flags.StringVar(&f.zone, "zone", "", "Zone to make GCE resources in.")
	flags.StringVar(&f.project, "project", "", "Project to make GCE resources in.")
	if f.labels == nil {
		f.labels = newMapVar()
	}
	flags.Var(f.labels, "labels", "Image labels to apply to the result image. Format is "+
		"'key1=value1,key2=value2,...'. Example: -labels=hello=world,foo=bar")
	if f.licenses == nil {
		f.licenses = &listVar{}
	}
	flags.Var(f.licenses, "licenses", "Image licenses to apply to the result image. Format is "+
		"'license1,license2,...' or '-licenses=license1 -licenses=license2'.")
	flags.BoolVar(&f.inheritLabels, "inherit-labels", false, "Indicates if the result image should inherit labels "+
		"from the source image. Labels specified through the '-labels' flag take precedence over inherited "+
		"labels.")
	flags.StringVar(&f.oemSize, "oem-size", "", "Size of the new OEM partition, "+
		"can be a number with unit like 10G, 10M, 10K or 10B, "+
		"or without unit indicating the number of 512B sectors.")
	flags.IntVar(&f.diskSize, "disk-size-gb", 0, "The disk size to use when creating the image in GB. Value of '0' "+
		"indicates the default size.")
	flags.DurationVar(&f.timeout, "timeout", time.Hour, "Timeout value of the image build process. Must be formatted "+
		"according to Golang's time.Duration string format.")
}

func (f *FinishImageBuild) validate() error {
	// The default size of the OEM partition in a COS image is assumed to be 16MB.
	const defaultOEMSizeMB = 16
	if f.oemSize != "" {
		oemSizeBytes, err := partutil.ConvertSizeToBytes(f.oemSize)
		if err != nil {
			return fmt.Errorf("invalid format of oem-size: %q, error msg:(%v)", f.oemSize, err)
		}
		if oemSizeBytes < (defaultOEMSizeMB << 20) {
			return fmt.Errorf("oem-size must be at least %dM", defaultOEMSizeMB)
		}
	}
	switch {
	case f.imageName == "" && f.imageSuffix == "":
		return fmt.Errorf("one of 'image-name' or 'image-suffix' must be set")
	case f.imageName != "" && f.imageSuffix != "":
		return fmt.Errorf("'image-name' and 'image-suffix' are mutually exclusive")
	case f.deprecateOld && f.imageFamily == "":
		return fmt.Errorf("'deprecate-old-images' can only be used if 'image-family' is set")
	case f.oldImageTTLSec != 0 && !f.deprecateOld:
		return fmt.Errorf("'old-image-ttl' can only be used if 'deprecate-old-images' is set")
	case f.zone == "":
		return fmt.Errorf("'zone' must be set")
	case f.project == "":
		return fmt.Errorf("'project' must be set")
	default:
		return nil
	}
}

func (f *FinishImageBuild) loadConfigs(files *fs.Files) (*config.Image, *config.Build, *config.Image, *provisioner.Config, error) {
	sourceImageConfig := &config.Image{}
	if err := config.LoadFromFile(files.SourceImageConfig, sourceImageConfig); err != nil {
		return nil, nil, nil, nil, err
	}
	imageName := f.imageName
	if f.imageSuffix != "" {
		imageName = sourceImageConfig.Name + f.imageSuffix
	}
	buildConfig := &config.Build{}
	if err := config.LoadFromFile(files.BuildConfig, buildConfig); err != nil {
		return nil, nil, nil, nil, err
	}
	buildConfig.Project = f.project
	buildConfig.Zone = f.zone
	buildConfig.DiskSize = f.diskSize
	buildConfig.Timeout = f.timeout.String()
	provConfig := &provisioner.Config{}
	if err := config.LoadFromFile(files.ProvConfig, provConfig); err != nil {
		return nil, nil, nil, nil, err
	}
	provConfig.BootDisk.OEMSize = f.oemSize
	outputImageConfig := config.NewImage(imageName, f.imageProject)
	outputImageConfig.Labels = f.labels.m
	outputImageConfig.Licenses = f.licenses.l
	outputImageConfig.Family = f.imageFamily
	return sourceImageConfig, buildConfig, outputImageConfig, provConfig, nil
}

func hasSealOEM(provConfig *provisioner.Config) bool {
	for _, s := range provConfig.Steps {
		if s.Type == "SealOEM" {
			return true
		}
	}
	return false
}

func validateOEM(buildConfig *config.Build, provConfig *provisioner.Config) error {
	// The default size of a COS image (imgSize) is assumed to be 10GB.
	const imgSize uint64 = 10
	// If auto-update is disabled, 2046MB will be reclaimed.
	// The size of sda3 is 2GB. We don't want to delete the partition,
	// so we need to leave some space in the sda3. And `sfdisk --move-data`
	// in some situations requires 1MB free space in the moving direction.
	// Therefore, leaving 2MB after the start of sda3 is a safe choice.
	// Also, this will make sure the start point of the next partition is
	// 4K aligned.
	const reclaimedMB uint64 = 2046
	const reclaimedBytes uint64 = reclaimedMB << 20
	var sizeError error
	var oemSizeBytes uint64
	var err error
	if !hasSealOEM(provConfig) {
		if provConfig.BootDisk.OEMSize == "" {
			return nil
		}
		// no need to seal the OEM partition.
		// If the OEM partition is to be extended, the following must be true:
		// disk-size >= imgSize + oem-size - reclaimed-size.
		if provConfig.BootDisk.ReclaimSDA3 {
			sizeError = fmt.Errorf("'disk-size-gb' must be at least 'oem-size'- reclaimed space "+
				"(%dMB) + image size (%dGB)", reclaimedMB, imgSize)
		} else {
			sizeError = fmt.Errorf("'disk-size-gb' must be at least 'oem-size' + image size (%dGB)", imgSize)
		}
		oemSizeBytes, err = partutil.ConvertSizeToBytes(provConfig.BootDisk.OEMSize)
		if err != nil {
			return fmt.Errorf("invalid format of oem-size: %q, error msg:(%v)", provConfig.BootDisk.OEMSize, err)
		}
	} else {
		// `seal-oem` will automatically disable auto-update and reclaim sda3.
		if provConfig.BootDisk.OEMSize == "" {
			// If need to seal OEM partition and the oem-size is not set,
			// assume the OEM fs size is 16M as it is in a COS image,
			// and the OEM partition size is doubled to 32M.
			// It will use space reclaimed from sda3.
			provConfig.BootDisk.OEMSize = "32M"
			provConfig.BootDisk.OEMFSSize4K = 4096
			return nil
		}
		// need extra space to seal the OEM partition.
		// The OEM partition size should be doubled to store the
		// hash tree of dm-verity. The following must be true:
		// disk-size >= imgSize + oem-size x 2 - reclaimed-size.
		sizeError = fmt.Errorf("'disk-size-gb' must be at least 'oem-size' x 2 - reclaimed space "+
			"(%dMB) + image size (%dGB)", reclaimedMB, imgSize)

		oemSizeBytes, err = partutil.ConvertSizeToBytes(provConfig.BootDisk.OEMSize)
		if err != nil {
			return fmt.Errorf("invalid format of oem-size: %q, error msg:(%v)", provConfig.BootDisk.OEMSize, err)
		}
		provConfig.BootDisk.OEMFSSize4K = oemSizeBytes >> 12
		// double the oem size.
		oemSizeBytes <<= 1
	}
	// Since we allow user input like "500M", and the "resize-disk" API can only take GB as input,
	// the oem-size is rounded up to GB to make sure there is enough space.
	// If the auto-update is disabled, space in sda3 will be reclaimed and used.
	// Extra space will be taken by the stateful partition.
	oemSizeReclaimBytes := oemSizeBytes
	if provConfig.BootDisk.ReclaimSDA3 {
		if oemSizeReclaimBytes <= reclaimedBytes {
			oemSizeReclaimBytes = 0
		} else {
			oemSizeReclaimBytes -= reclaimedBytes
		}
	}
	oemSizeGB, err := partutil.ConvertSizeToGBRoundUp(strconv.FormatUint(oemSizeReclaimBytes, 10) + "B")
	if err != nil {
		return fmt.Errorf("invalid format of oem-size: %q, error msg:(%v)", provConfig.BootDisk.OEMSize, err)
	}
	//  if no disk-size-gb input, assume the default image size to be 10GB.
	var diskSize uint64 = imgSize
	if buildConfig.DiskSize != 0 {
		diskSize = (uint64)(buildConfig.DiskSize)
	}
	if diskSize < imgSize+oemSizeGB {
		return sizeError
	}
	// Shrink OEM size input (rounded down) by 1MB to deal with cases
	// where disk size is 1MB smaller than needed.
	// This will take 1MB from the hash tree part (the second half)
	// of the OEM partition if seal-oem is set. Otherwise, it will
	// take 1MB from user data space of the OEM partition.
	// For example oem-size=1G, disk-size-gb=11, seal-oem not set.
	// Or oem-size=1G, disk-size-gb=11, seal-oem set.
	// In those cases the disk size is not large enough without shrinking
	// the OEM partition size by 1MB.
	provConfig.BootDisk.OEMSize = strconv.FormatUint((oemSizeBytes>>20)-1, 10) + "M"
	return nil
}

func update(dst, src map[string]string) {
	for k, v := range src {
		if _, ok := dst[k]; !ok {
			dst[k] = v
		}
	}
}

// Execute implements subcommands.Command.Execute. It gathers image configuration parameters
// and creates a GCE image.
func (f *FinishImageBuild) Execute(ctx context.Context, flags *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	if flags.NArg() != 0 {
		flags.Usage()
		return subcommands.ExitUsageError
	}
	files := args[0].(*fs.Files)
	defer files.CleanupAllPersistent()
	svc, gcsClient, err := args[1].(ServiceClients)(ctx, false)
	if err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}
	defer gcsClient.Close()
	if err := f.validate(); err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}
	sourceImage, buildConfig, outputImage, provConfig, err := f.loadConfigs(files)
	if err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}
	if err := validateOEM(buildConfig, provConfig); err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}
	exists, err := gce.ImageExists(svc, outputImage.Project, outputImage.Name)
	if err != nil {
		log.Println(err)
		return subcommands.ExitFailure
	}
	if exists {
		log.Printf("Result image %s already exists in project %s. Exiting.\n", outputImage.Name, outputImage.Project)
		return subcommands.ExitSuccess
	}
	if f.inheritLabels {
		image, err := svc.Images.Get(sourceImage.Project, sourceImage.Name).Do()
		if err != nil {
			log.Println(err)
			return subcommands.ExitFailure
		}
		update(outputImage.Labels, image.Labels)
	}
	if err := preloader.BuildImage(ctx, gcsClient, files, sourceImage, outputImage, buildConfig, provConfig); err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			log.Printf("command failed: %s. See stdout logs for details", err)
			return subcommands.ExitFailure
		}
		log.Println(err)
		return subcommands.ExitFailure
	}
	if f.deprecateOld {
		if err := gce.DeprecateInFamily(ctx, svc, outputImage, f.oldImageTTLSec); err != nil {
			log.Printf("deprecating images failed: %s", err)
			return subcommands.ExitFailure
		}
	}
	return subcommands.ExitSuccess
}
