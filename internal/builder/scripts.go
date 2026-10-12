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

// RemovePackageScript deletes one package file from the repository and regenerates the database.
//
//go:embed remove_package.sh
var RemovePackageScript string

// UpdateRepoDbScript rebuilds the pacman database so it lists the newest version of every package.
//
//go:embed update_repo_db.sh
var UpdateRepoDbScript string

// OpenZFSKeys are the public keys that sign OpenZFS releases (the validpgpkeys of the AUR zfs-utils
// PKGBUILD). Shipped with the scripts so source verification does not depend on a keyserver.
//
//go:embed openzfs-keys.asc
var OpenZFSKeys string
