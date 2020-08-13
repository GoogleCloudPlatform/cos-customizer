# Image Customizer for Container-Optimized OS from Google

Note: This is not an official Google product.

The COS Customizer is a tool for creating customized Container-Optimized OS
images. It uses
[Daisy](https://github.com/GoogleCloudPlatform/compute-image-tools/tree/master/daisy)
to create a COS VM instance, load data onto the instance, and create a disk
image from the modified instance.

Currently, the COS Customizer is intended to be run as part of a
[Google Cloud Build](https://cloud.google.com/cloud-build/) workflow as a
sequence of Google Cloud Build build steps. No other usage mode is currently
supported.

*   [Accessing the cos-customizer container image](#accessing-the-cos-customizer-container-image)
*   [Quick Start](#quick-start)
    *   [Minimal example](#minimal-example)
*   [Build Steps](#build-steps)
    *   [Required build steps](#required-build-steps)
        *   [The start-image-build step](#the-start-image-build-step)
        *   [The finish-image-build step](#the-finish-image-build-step)
    *   [Optional build steps](#optional-build-steps)
        *   [run-script](#run-script)
        *   [install-gpu](#install-gpu)
        *   [seal-oem](#seal-oem)
        *   [disable-auto-update](#disable-auto-update)

## Accessing the cos-customizer container image

The container image is available at `gcr.io/cos-cloud/cos-customizer`.
Alternatively, it can be built from source using [Bazel](https://bazel.build/).
To build COS customizer and load the image into Docker, run:

    $ bazel run :cos_customizer -- --norun

The COS Customizer docker image will then be available in Docker as
`bazel:cos_customizer`.

## Quick Start

The COS Customizer is intended to be run as a sequence of steps in a Google
Cloud Build workflow. It is implemented and distributed as a Docker container.
Each subcommand of the COS Customizer implements a Google Cloud Build build
step. Two of these steps need to be present for every image build, and the rest
of the steps are optional steps that can be used for customizing a COS image.

The required build steps are the `start-image-build` and `finish-image-build`
steps. The `start-image-build` step initializes local state for the image build,
and the `finish-image-build` step performs the image building operation with
Daisy.

Example optional build steps are `run-script`, `install-gpu`, `seal-oem` and
`disable-auto-update`.  
`run-script`allows users to customize an image by running a script.  
`install-gpu` allows users to install GPU drivers using the
[COS GPU installer](https://github.com/GoogleCloudPlatform/cos-gpu-installer).  
`seal-oem` allows users to setup a verified read-only OEM partition. It will be 
verified when the VM boots and when the data inside are accessed.  
`disable-auto-update` allows users to disable the auto-update service. And it
will reclaim the disk space of the unused root partition.

### Minimal example

Here is a minimal Google Cloud Build workflow demonstrating usage of the COS
Customizer. It customizes the image `cos-stable-68-10718-86-0` by running the
script `preload.sh`. This results in an image with the custom file
`/var/lib/hello`.

    $ cat preload.sh
    echo "Hello, World!" > /var/lib/hello
    $ cat cloudbuild.yaml
    steps:
    - name: 'gcr.io/cos-cloud/cos-customizer'
      args: ['start-image-build',
             '-image-name=cos-stable-68-10718-86-0',
             '-image-project=cos-cloud',
             '-gcs-bucket=${PROJECT_ID}_cloudbuild',
             '-gcs-workdir=image-build-$BUILD_ID']
    - name: 'gcr.io/cos-cloud/cos-customizer'
      args: ['run-script',
             '-script=preload.sh']
    - name: 'gcr.io/cos-cloud/cos-customizer'
      args: ['finish-image-build',
             '-zone=us-west1-b',
             '-project=$PROJECT_ID',
             '-image-name=my-custom-image',
             '-image-project=$PROJECT_ID']
    timeout: '1500s'
    $ gcloud builds submit --config=cloudbuild.yaml .

## Build Steps

The COS Customizer is different from typical Google Cloud Build build steps.
Most build steps, like the `gcr.io/cloud-builders/gcloud` build step, are
single-purpose container images that are capable of being useful when run in
isolation. The COS Customizer is not one of these build steps.

The COS Customizer is a container image that provides a collection of Google
Cloud Build build steps that are intended to be used together. When run in
sequence as part of a Google Cloud Build workflow, these build steps create a
Compute Engine disk image.

Each build step is invoked as a subcommand of the COS Customizer container
image; for example, usage of the `run-script` build step works as follows:

    ...
    - name: 'gcr.io/cos-cloud/cos-customizer'
      args: ['run-script',
             '-script=preload.sh']
    ...

### Required build steps

Two build steps are required for each image build operation; the
`start-image-build` step and the `finish-image-build` step.

#### The start-image-build step

The primary purpose of this step is to initialize the image build process. It
only initializes local state in the Google Cloud Build builder. It does not
create any cloud resources. It must run before all of the other steps in the
image build process, and it must only be run once. It takes the following flags:

`-build-context`: A path to a file or directory that should be relative to the
default Google Cloud Build working directory. Defaults to `.`. The contents of
this path will be copied to the builder VM in a temporary directory. All scripts
specified by a `run-script` step will execute with this directory as a working
directory. For example, suppose that the source directory provided to Google
Cloud Build looked like this:

    .
    ├── lib
    │   └── mylib.sh
    └── main.sh

If `-build-context` is set to `.`, this directory structure will be copied to
the builder VM and will be the working directory for all specified `run-script`
steps. If a `run-script` step runs the script `main.sh`, `main.sh` will have
access to `lib/mylib.sh`. However, suppose `-build-context` is set to `lib`;
then, a `run-script` step that specifies `main.sh` will fail, since `main.sh`
won't be included in the working directory on the builder VM. Specifying
`mylib.sh` in a `run-script` step would be valid in this case though.

`-gcs-bucket`: A GCS bucket to use for scratch space. Optional build steps are
free to use this bucket for scratch space. Normally, it's expected that only
`finish-image-build` will use this GCS bucket. `finish-image-build` uses this
GCS bucket for transferring binary blobs to the builder VM.

`-gcs-workdir`: A directory in the aforementioned GCS bucket that will be used
for scratch space.

`-image-project`: The Google Cloud Platform (GCP) project that contains the
source image; that is, the image to customize.

`-image-name`: The name of the source image. Mutually exclusive with
`-image-milestone` and `-image-family`.

`-image-milestone`: The milestone of the source image. If `-image-milestone` is
specified and `-image-project` is set to `cos-cloud`, the `start-image-build`
step will resolve the source image by finding the latest image in `cos-cloud` on
the specified milestone. An example value for this field is `69`. Mutually
exclusive with `-image-name` and `-image-family`.

`-image-family`: The family of the source image. If `-image-family` is
specified, the `start-image-build` step will resolve the source image by finding
the latest active image in the specified image family. This is done using Google
Compute Engine's `getFromFamily` API. Mutually exclusive with `-image-name` and
`-image-milestone`.

An example `start-image-build` step looks like the following:

    - name: 'gcr.io/cos-cloud/cos-customizer'
      args: ['start-image-build',
             '-image-name=cos-stable-68-10718-86-0',
             '-image-project=cos-cloud',
             '-gcs-bucket=${PROJECT_ID}_cloudbuild',
             '-gcs-workdir=image-build-$BUILD_ID']

#### The finish-image-build step

The primary purpose of this step is to execute the steps specified in the image
build process. This step creates a builder VM, runs configured scripts on it,
and creates a disk image from the VM. It must run after all of the other steps
in an image build process. This step will clean up the local state stored by
previous COS Customizer steps; a new image build process can be started after a
`finish-image-build` step. It takes the following flags:

`-image-project`: The GCP project that should contain the output image.

`-image-name`: The name of the output image. Mutually exclusive with
`-image-suffix`.

`-image-suffix`: Construct the name of the output image by appending the
specified suffix to the name of the input image. Mutually exclusive with
`-image-name`.

`-image-family`: An image family to assign the output image to.

`-deprecate-old-images`: If present, the image build process will deprecate all
of the old images in the output image's image family. Can only be specified if
`-image-family` is specified.

`-old-image-ttl`: Time-to-live in seconds to apply to images deprecated by
`-deprecate-old-images`. Configures the "deleted" field of the image's
deprecation status to be this many seconds after the image is deprecated. Can
only be used if `-deprecate-old-images` is also given.

`-zone`: The GCE zone in which to perform the image building operation. This is
an important consideration when installing GPU drivers on the image, since
installing GPU drivers requires that GPU quota is available in this zone.

`-project`: The GCP project to use for the image building operation.

`-labels`: Key-value pairs to apply to the output image as image labels.
Example: `-labels=cos_image=true,milestone=65`

`-licenses`: A list of licenses to apply to the output image. License names must
be formatted as `projects/{project}/global/licenses/{license}`. Example:
`-licenses=projects/cos-cloud/global/licenses/cos`

`-inherit-labels`: If present, the output image will be assigned the exact same
image labels present on the source image. The labels specified by the `-labels`
flag take precedence over labels assigned with this flag.

`-disk-size-gb`: The disk size in GB to use when creating the image.
This value should never be smaller than 10 (the default size of a COS image).
If `-oem-size` is set,  the lower limit of `-disk-size-gb` is as shown in the 
following table. The larger one of the value in the table and 10 is 
effective. See section `-oem-size`,
[seal-oem](#seal-oem) and [disable-auto-update](#disable-auto-update) for details.

| disk-size-gb-lower-limit |        no seal-oem       |           seal-oem           |
|:------------------------:|:------------------------:|:----------------------------:|
|  no disable-auto-update  |      10GB + oem-size     | 10GB + oem-size x 2 - 2046MB |
|    disable-auto-update   | 10GB + oem-size - 2046MB | 10GB + oem-size x 2 - 2046MB | 

Note that if `seal-oem` is run without specifying `-oem-size`, the lower limit of
`-disk-size-gb` will be 10.

`-oem-size`: The file system size of the extended OEM partition with unit 
`G`,`M`,`K` or `B`. 
If no unit is provided, it will be parsed as the number of sectors of 512 Bytes.
Since the default size of the OEM partition in a COS image is assumed to be 16MB, 
this value must be no smaller than 16MB, otherwise the build will fail. 
Make sure the disk size is large enough if this flag is used to extend the OEM partition.
If the `seal-oem` or `disable-auto-update` is run, the OEM partition will firstly
use the reclaimed space.
See section `-disk-size-gb` for the limits of the disk size value.
Example: `-oem-size=500M`

Note that this feature is supported by COS versions higher than milestone 73 (included).

`-timeout`: Timeout value of this step. Must be formatted according to Golang's
time.Duration string format. Defaults to "1h0m0s". Keep in mind that this timeout
value is different from the overall Cloud Build workflow timeout value, which is
set at the Cloud Build workflow level. If this timeout value expires, resources
created during the image build process will be properly cleaned up. If the
overall Cloud Build workflow timeout expires, the task will be cancelled without
any opportunity to clean up resources.

An example `finish-image-build` step looks like the following:

    - name: 'gcr.io/cos-cloud/cos-customizer'
      args: ['finish-image-build',
             '-zone=us-west1-b',
             '-project=$PROJECT_ID',
             '-image-name=my-custom-image',
             '-image-project=$PROJECT_ID']

### Optional build steps

The rest of the build steps provided by COS Customizer are optional; if they are
not included, the image build will run successfully, but will generate an image
that is identical to the source image. Optional build steps are used to make
meaningful changes to an image.

#### run-script

The `run-script` build step configures the image build to run a script on the
builder VM. If multiple `run-script` steps are given, the scripts specified by
each step will run in the same order in which the `run-script` steps were given.
It takes the following flags:

`-script`: A path to the script to run. The path should be relative to the root
of the build context provided in `start-image-build`.

`-env`: Key-value pairs indicating environment variables to provide to the
script when it is run. Example: `-env=RELEASE=1,FOO=bar`

An example `run-script` step looks like the following:

    - name: 'gcr.io/cos-cloud/cos-customizer'
      args: ['run-script',
             '-script=preload.sh']

#### install-gpu

The `install-gpu` build step configures the image build to install GPU drivers
on the builder VM. GPU drivers are installed using the
[COS GPU installer](https://github.com/GoogleCloudPlatform/cos-gpu-installer).
In addition to installing GPU drivers, the `install-gpu` step installs a script
named `setup_gpu.sh` in the GPU driver install directory. _In order to use the
installed GPU drivers, this script must be run every time the system boots_. It
should be executed as part of a startup script or cloud config. `install-gpu`
takes the following flags:

`-version`: The GPU driver version to install. Currently, we only support
installing Tesla drivers that are present in the
[nvidia-drivers-us-public GCS bucket](https://console.cloud.google.com/storage/browser/nvidia-drivers-us-public).
The set of supported drivers can be found by running the `install-gpu` step
independently on your local machine with the `-get-driver-version` flag.
Example: `-version=396.26`

`-get-driver-version`: Prints out the list of supported driver versions to
stdout and exits. If this flag is provided, the build step doesn't do anything
meaningful; it only prints the list of supported driver versions. It is not
intended to be used in a Google Cloud Build workflow; it is meant to be run
independently for users to easily see the set of supported driver versions.

`-md5sum`: If you have the md5sum of the driver you want to install, you can
provide it here and the COS GPU installer will verify the driver with this
md5sum.

`-install-dir`: The directory on the image to install GPU drivers to. The
`setup_gpu.sh` script will also be installed in this directory. Make sure to
choose a directory that will persist across reboots; for the most part, this
means a subdirectory of `/var` or `/home`.

`-gpu-type`: The type of GPU to use to verify correct installation of GPU
drivers. The valid values here are nvidia-tesla-k80, nvidia-tesla-p100, and
nvidia-tesla-v100. This value has no impact on the drivers that are installed on
the image; it is only used when verifying that the driver installation
succeeded. Make sure that the zone you are running the image build in has quota
for a GPU of this type.

An example `install-gpu` step looks like the following:

    - name: 'gcr.io/cos-cloud/cos-customizer'
      args: ['install-gpu',
             '-version=396.26']

Note that when using an image customized with `install-gpu`, the hosted docker
container should be set to run in privileged mode so that it has access to the
GPU device on the host machine.

#### seal-oem

The `seal-oem` build step utilizes `dm-verity` to verify the data in the OEM
partition when the system boots and when data are accessed. 
If the verification fails, the system will refuse to boot or will panic. 
This step takes no flags and needs to be run after any step 
that makes changes to the OEM partition (`/dev/sda8` or `/usr/share/oem`).

If this step is run, the size of the OEM partition will be doubled to store
the hash tree for verification in the second half of the partition.
If `-oem-size` in `finish-image-build` step is not set, the file system 
size of the OEM partition will be assumed to be the same as the default size, 
16MB. And the size of the OEM partition will be doubled to 32MB.

The auto-update service is automatically disabled in this step. So it is not 
necessary to run the `disable-auto-update` step explicitly. This will reclaim
the unused space and the OEM partition will firstly use the reclaimed space.
See section `-disk-size-gb` for the limits of the disk size value. If the 
disk size is not large enough, the build will fail.

After running this build step, the OEM partition will not be automatically
mounted when the system boots.   
`sudo mount /dev/dm-1 /usr/share/oem` should be  added to 
`startup script` or `cloud init` to mount the OEM partition.

Note that this feature is supported by COS versions higher than milestone 73 (included).

#### disable-auto-update

The `disable-auto-update` build step modifies the kernel commandline to disable
the auto-update serive. This step takes no flags.

The root partition that is used by auto-update service will not be needed anymore,
so the disk space (2046MB) of that partition will be reclaimed. The reclaimed
space will be used by the OEM partition if extended and the stateful partition.

Note that this feature is supported by COS versions higher than milestone 73 (included).

# Contributor Docs

## Releasing

To release a new version of COS Customizer, tag the commit you want to release
with the date in the form of `vYYYYMMDD`. This will trigger a Cloud Build job to
build and release the container image.
