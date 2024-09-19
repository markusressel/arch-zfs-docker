#!/bin/bash

PACKAGE_DIR="/packages"
CUSTOM_PACMAN_REPO_PATH="/home/build/repo"
mkdir -p "$CUSTOM_PACMAN_REPO_PATH"
CUSTOM_PACMAN_REPO_DB_PATH="$CUSTOM_PACMAN_REPO_PATH/custom.db.tar.zst"

function update_custom_repo () {
    repo-add "$CUSTOM_PACMAN_REPO_DB_PATH" /home/build/repo/*.pkg.tar
}

############
# zfs-util #
############

gpg --keyserver keys.gnupg.net --recv-keys 6AD860EED4598027
git clone --depth 1 --quiet https://aur.archlinux.org/zfs-utils.git
pushd zfs-utils
  # build zfs-utils and install it within the container
  makepkg --install --syncdeps --clean --rmdeps --noconfirm

  cp ./*.pkg.tar "$CUSTOM_PACMAN_REPO_PATH"
  cp ./*.pkg.tar "$PACKAGE_DIR"
popd

update_custom_repo
# sync local package repositores
# sudo pacman --noconfirm -Syy

#############
# zfs-linux #
#############

git clone --depth 1 --quiet https://aur.archlinux.org/zfs-linux.git
pushd zfs-linux
  PKGVER=$(grep "^pkgver=" ../zfs-utils/PKGBUILD | awk -F'=' '{print $2}')
  SHA256SUM=$(grep "^sha256sums=" ../zfs-utils/PKGBUILD | awk -F"'" '{print $2}')
  # update PKGBUILD to match the earlier build zfs-utils version
  /usr/local/bin/update_pkgbuild.sh "$PKGVER" "$SHA256SUM"
  # create certificate for signing the package lateron
  openssl req -new -nodes -utf8 -sha512 -days 36500 -batch -x509 -config x509.genkey -outform DER -out signing_key.x509 -keyout signing_key.pem
  
  # build zfs-linux
  makepkg --syncdeps --clean --rmdeps --noconfirm
  
  cp ./*.pkg.tar "$CUSTOM_PACMAN_REPO_PATH"
  cp ./*.pkg.tar "$PACKAGE_DIR"
popd
