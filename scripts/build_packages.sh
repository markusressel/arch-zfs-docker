#!/bin/bash

# 1. Force refresh the keyring before ANY pacman operation
sudo pacman -Sy --noconfirm archlinux-keyring
sudo pacman-key --init
sudo pacman-key --populate archlinux

VARIANT=""
if [[ "$LTS" != "" ]]; then
  VARIANT="-lts"
fi

PACKAGE_DIR="/packages"
CUSTOM_PACMAN_REPO_PATH="/home/build/repo"
mkdir -p "$CUSTOM_PACMAN_REPO_PATH"
CUSTOM_PACMAN_REPO_DB_PATH="$CUSTOM_PACMAN_REPO_PATH/custom.db.tar.zst"

# If the database doesn't exist, create an empty one
if [ ! -f "$CUSTOM_PACMAN_REPO_DB_PATH" ]; then
    touch "$CUSTOM_PACMAN_REPO_PATH/placeholder"
    repo-add "$CUSTOM_PACMAN_REPO_DB_PATH" "$CUSTOM_PACMAN_REPO_PATH/placeholder"
    rm "$CUSTOM_PACMAN_REPO_PATH/placeholder"
fi

# Now Pacman can sync without errors
sudo pacman --noconfirm -Syy

function update_custom_repo () {
    repo-add "$CUSTOM_PACMAN_REPO_DB_PATH" /home/build/repo/*.pkg.tar
}

############
# zfs-util #
############

gpg --keyserver keys.gnupg.net --recv-keys 6AD860EED4598027 0AB9E991C6AF658B
git clone --depth 1 --quiet https://aur.archlinux.org/zfs-utils.git
pushd zfs-utils || exit
  # build zfs-utils and install it within the container
  makepkg --install --syncdeps --clean --rmdeps --noconfirm

  cp ./*.pkg.tar "$CUSTOM_PACMAN_REPO_PATH"
  cp ./*.pkg.tar "$PACKAGE_DIR"
popd || exit

update_custom_repo
# sync local package repositores
# sudo pacman --noconfirm -Syy

#############
# zfs-linux #
#############

git clone --depth 1 --quiet https://aur.archlinux.org/zfs-linux$VARIANT.git
pushd "zfs-linux$VARIANT" || exit
  PKGVER=$(grep "^pkgver=" ../zfs-utils/PKGBUILD | awk -F'=' '{print $2}')
  SHA256SUM=$(grep "^sha256sums=" ../zfs-utils/PKGBUILD | awk -F"'" '{print $2}')
  # update PKGBUILD to match the earlier build zfs-utils version
  /usr/local/bin/update_pkgbuild.sh "$PKGVER" "$SHA256SUM"
  # create certificate for signing the package lateron
  openssl req -new -nodes -utf8 -sha512 -days 36500 -batch -x509 \
    -subj "/C=US/ST=State/L=City/O=Organization/OU=Unit/CN=Regenerated-ZFS-Key" \
    -outform DER -out signing_key.x509 -keyout signing_key.pem
  
  # build zfs-linux(-lts)
  makepkg --syncdeps --clean --rmdeps --noconfirm
  
  cp ./*.pkg.tar "$CUSTOM_PACMAN_REPO_PATH"
  cp ./*.pkg.tar "$PACKAGE_DIR"
popd || exit

update_custom_repo