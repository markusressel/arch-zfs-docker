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

// UpdateRepoDbScript rebuilds the pacman database so it lists the newest version of every package.
//
//go:embed update_repo_db.sh
var UpdateRepoDbScript string
