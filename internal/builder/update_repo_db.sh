#!/bin/bash
# Rebuilds the pacman database from the packages in a repository directory.
#
# Old package files are kept on disk so users on an older kernel can still `pacman -U` a matching
# build, but a pacman database can only list ONE version per package name, and it must be the
# newest. repo-add alone would let the most recently added package win, even if it is older
# (e.g. a rebuild for an older kernel), which would make newer systems miss their upgrade.
#
# "Newest" means newest KERNEL first, then newest ZFS version: zfs-linux depends on linux=<kernel>
# exactly, so the database must follow the kernel Arch ships. Otherwise a higher ZFS version built
# for an older kernel would hide the build for the current kernel and block `pacman -Syu`.
#
# Usage: update_repo_db.sh <repo-dir> <repo-name>
set -e

REPO_DIR="${1:?repo dir required}"
REPO_NAME="${2:?repo name required}"

cd "$REPO_DIR"
shopt -s nullglob

# Serialize with concurrent jobs: repo-add fails on a held lock instead of waiting.
exec 9>"$REPO_DIR/.repo-db.lock"
flock 9

# Succeeds if version "$1" (<pkgver>-<pkgrel>) is newer than "$2".
# Kernel module packages have pkgver <zfsver>_<kernelver>; compare the kernel part first.
is_newer() {
  local a="$1" b="$2"
  if [[ $a == *_* && $b == *_* ]]; then
    local c
    c=$(vercmp "${a#*_}" "${b#*_}")
    if [ "$c" -ne 0 ]; then
      [ "$c" -gt 0 ]
      return
    fi
    a="${a%%_*}"
    b="${b%%_*}"
  fi
  [ "$(vercmp "$a" "$b")" -gt 0 ]
}

declare -A newest_file newest_ver
for f in *.pkg.tar.zst; do
  # <name>-<pkgver>-<pkgrel>-<arch>.pkg.tar.zst (pkgver never contains a hyphen)
  base="${f%.pkg.tar.zst}"
  rest="${base%-*}"       # strip arch
  rel="${rest##*-}"
  rest="${rest%-*}"       # strip pkgrel
  ver="${rest##*-}"
  name="${rest%-*}"
  full="${ver}-${rel}"
  if [ -z "${newest_ver[$name]}" ] || is_newer "$full" "${newest_ver[$name]}"; then
    newest_file[$name]="$f"
    newest_ver[$name]="$full"
  fi
done

if [ ${#newest_file[@]} -eq 0 ]; then
  echo "==> No packages found in $REPO_DIR, nothing to do."
  exit 0
fi

# Build under a temporary name and move into place, so clients never see a missing database.
tmp="tmp-${REPO_NAME}"
rm -f "${tmp}".db* "${tmp}".files*
repo-add "${tmp}.db.tar.zst" "${newest_file[@]}"
mv -f "${tmp}.db.tar.zst" "${REPO_NAME}.db.tar.zst"
mv -f "${tmp}.files.tar.zst" "${REPO_NAME}.files.tar.zst"
ln -sf "${REPO_NAME}.db.tar.zst" "${REPO_NAME}.db"
ln -sf "${REPO_NAME}.files.tar.zst" "${REPO_NAME}.files"
rm -f "${tmp}".db* "${tmp}".files*

echo "==> Database ${REPO_NAME}.db.tar.zst lists:"
for name in "${!newest_file[@]}"; do
  echo "    ${newest_file[$name]}"
done | sort
