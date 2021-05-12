package provisioner

const anthosInstallerInstallTemplateScript = `#!/bin/bash

set -o errexit

export ANTHOS_INSTALLER_VERSION={{.AnthosInstallerVersion}}
export PACKAGE_SPEC_DIR={{.PkgSpecDir}}
export TOP_WORK_DIR={{.TopWorkDir}}
export ANTHOS_INSTALLER_DIR={{.AnthosInstallerDir}}
export ANTHOS_INSTALLER_RELEASE_BUCKET=cos-anthos-builds-us

install_anthos_installer() {
 gsutil cp gs://${ANTHOS_INSTALLER_RELEASE_BUCKET}/anthos_installer-${ANTHOS_INSTALLER_VERSION} ${ANTHOS_INSTALLER_DIR}/anthos_installer.tar.gz
 tar -xvf ${ANTHOS_INSTALLER_DIR}/anthos_installer.tar.gz
}

install_packages() {
	${ANTHOS_INSTALLER_DIR}/anthos_installer install -pkg-spec-dir=${PACKAGE_SPEC_DIR} -work-dir=${TOP_WORK_DIR}
}

cleanup() {
	rm -rf ${ANTHOS_INSTALLER_DIR}
}

main() {
	install_anthos_installer
	install_packages
	cleanup
}


main
`
