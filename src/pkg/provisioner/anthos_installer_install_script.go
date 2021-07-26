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

package provisioner

const anthosInstallerInstallTemplateScript = `#!/bin/bash

set -o errexit

export PACKAGE_SPEC_DIR={{.PkgSpecURL}}
export ANTHOS_INSTALLER_DIR={{.AnthosInstallerDir}}
export ANTHOS_INSTALLER_VERSION={{.AnthosInstallerVersion}}
export BUILD_CONTEXT_DIR={{.BuildContext}}
export BIN_DIR={{.AnthosInstallerDir}}/bin

install_anthos_installer() {
	tar -xvf ${ANTHOS_INSTALLER_DIR}/${ANTHOS_INSTALLER_VERSION} -C ${ANTHOS_INSTALLER_DIR}
	cp ${ANTHOS_INSTALLER_DIR}/anthos_installer ${BIN_DIR}/anthos_installer
}

install_packages() {
	sudo ${BIN_DIR}/anthos_installer install -pkgspec-url=${PACKAGE_SPEC_DIR} -build-contextdir=${BUILD_CONTEXT_DIR}
	echo "Successfully installed the packages"
}

main() {
	install_anthos_installer
	install_packages
}

main
`
