package builder

import _ "embed"

// CheckAndBuildScript is the main build entrypoint executed inside the builder container.
//
//go:embed check_and_build.sh
var CheckAndBuildScript string

// UpdatePkgbuildScript updates the AUR PKGBUILD with target versions and module paths.
//
//go:embed update_pkgbuild.sh
var UpdatePkgbuildScript string

// OpenZFSKeys are the public keys that sign OpenZFS releases (the validpgpkeys of the AUR zfs-utils
// PKGBUILD). Shipped with the scripts so source verification does not depend on a keyserver.
//
//go:embed openzfs-keys.asc
var OpenZFSKeys string
