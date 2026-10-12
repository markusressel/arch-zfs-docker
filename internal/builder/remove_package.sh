#!/bin/bash
# Removes one package file (and its signature) from the repository and regenerates the database.
#
# Environment: REPO_NAME, PACKAGE_FILE (plain file name), JOB_NAME
set -e

REPO_NAME="${REPO_NAME:?REPO_NAME required}"
PACKAGE_FILE="${PACKAGE_FILE:?PACKAGE_FILE required}"
JOB_NAME="${JOB_NAME:-remove-$(date +%s)}"
REPO_ROOT="${REPO_ROOT:-/repo}"
REPO_DIR="${REPO_ROOT}/${REPO_NAME}/x86_64"
LOG_DIR="${REPO_ROOT}/logs"

mkdir -p "$LOG_DIR"
exec > >(tee -a "${LOG_DIR}/${JOB_NAME}.log") 2>&1

case "$PACKAGE_FILE" in
  */* | .* | "" | *[!A-Za-z0-9@._+-]*)
    echo "ERROR: refusing unsafe package file name: $PACKAGE_FILE"
    exit 1
    ;;
  *.pkg.tar.zst | *.pkg.tar.xz | *.pkg.tar.gz) ;;
  *)
    echo "ERROR: not a package file: $PACKAGE_FILE"
    exit 1
    ;;
esac

cd "$REPO_DIR"
if [ ! -f "$PACKAGE_FILE" ]; then
  echo "==> $PACKAGE_FILE does not exist in $REPO_DIR (already removed?), refreshing the database anyway."
else
  echo "==> Removing $PACKAGE_FILE"
  rm -f -- "$PACKAGE_FILE" "${PACKAGE_FILE}.sig"
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
echo "==> Updating pacman repository database: ${REPO_NAME}.db.tar.zst"
"$SCRIPT_DIR/update_repo_db.sh" "$REPO_DIR" "$REPO_NAME"
echo "==> Done."
