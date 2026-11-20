# The builder

A build job runs in the `archlinux:base-devel` image. Everything it does is Go code in `internal/builder`, run as the
server's own binary in builder mode (`builder build`, `builder remove-package`); there are no shell scripts.

## Steps of a build

1. Initialize the pacman keyring, then `pacman -Sy archlinux-keyring`.
2. Pick the kernel: the newest one of Arch Linux, or the requested version from the Arch Linux Archive.
3. Stop if the repository has the package already (unless `FORCE_BUILD`).
4. Create the build user (makepkg refuses root), point pacman and makepkg at the caches on the volume, and install
   the tools with `pacman -Su --needed base-devel git pacman-contrib openssl curl`.
5. Install the kernel and its headers.
6. `1/2`: build `zfs-utils` from the AUR, or install it from the repository if it has that version.
7. `2/2`: build `zfs-linux` from the AUR, with its PKGBUILD adapted to the kernel and the OpenZFS version.
8. Copy the packages into the repository and update the database with `repo-add`.

In Kubernetes jobs, step 8 ends after the copy: the build runs in the init container `builder` and lists the packages
it published in `$HANDOFF_FILE` (an `emptyDir` of the pod), each with the SHA-256 of the copied bytes. The container
`index` (`builder index`) then signs exactly these packages with the key of `SIGNING_SECRET`, each from a private copy
checked against its hash, rebuilds the database with `repo-add --sign` and signs it. The
Docker runner and `builder update-repo-db` keep the unsigned database step.

Step 4 must stay `-Su`. With `-S` alone it is a partial upgrade, and every build fails when Arch Linux changes a
library the image depends on, as in November 2026 with openssl 4.

## Rules

- External tools (`pacman`, `makepkg`, `git`, `gpg`, `repo-add`, ...) run through `builder.Runner`. Commands as root
  and as the build user use separate runners.
- Tests replace the runners with fakes: `fakeSystem` in `build_test.go` records every command with its user, so the
  tests need neither root nor a container. Tests that need the real tools skip when they are missing.
- `internal/compat` reads the build log: the kernel and ZFS version lines and the line `==> 2/2: Building zfs-linux`.
  Keep these lines as they are; `TestBuildLogIsUnderstoodByTheCompatClassifier` checks them.
- Everything that downloads into the shared source cache (`makepkg --verifysource`, `updpkgsums`) holds the source
  cache lock. Builds run on the finished files afterwards.
- The OpenZFS tarball is checked against its detached signature (`validpgpkeys`, keys in
  `internal/builder/openzfs-keys.asc`). Never pass `--skippgpcheck` to makepkg. The checksums are recomputed for other
  OpenZFS versions, so the signature is the check that counts.
- The signing key must never be readable in the container that runs AUR code: the build user may use sudo, so it is
  root there. Only `index` mounts the Secret, and it runs the builder binary from its own volume, which `builder`
  cannot write (`TestRealClientJobPodKeepsTheKeyAwayFromTheBuild`).
- With a signer, `UpdateRepoDB` lists only packages whose signature verifies, and never signs a package it did not
  get from `IndexRepo`'s handoff list: signing what is on the disk would sign a swapped package. Without a signer it
  removes the database signatures, because pacman rejects a database whose signature does not match.
- The binary must be statically linked (`CGO_ENABLED=0`, set by the Justfile and the Dockerfile). The Docker runner
  uploads the server's executable into each build container and refuses a dynamically linked one.
- Versions are compared with `builder.Vercmp`, a port of pacman's `vercmp`. `TestVercmpMatchesPacman` keeps the two in
  agreement.

## Logs

The builder writes its output to stdout and appends it to `/repo/logs/<job>.log`. The server reads that file for the
web UI after the job is gone and for the classification of failures. A Job that retries appends every attempt to the
same file.
