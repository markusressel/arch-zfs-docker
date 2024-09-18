#!/bin/bash

WORKING_DIR="/package"
CUSTOM_PACMAN_REPO_DB_PATH="/home/build/custom.db.tar.zst"

# echo "Pacman Configuration:"
# cat /etc/pacman.conf

gpg --keyserver keys.gnupg.net --recv-keys 6AD860EED4598027
rm -rf zfs-utils
git clone --depth 1 --quiet https://aur.archlinux.org/zfs-utils.git
pushd zfs-utils
makepkg --install --syncdeps --clean --rmdeps --noconfirm
#makepkg --syncdeps --clean --rmdeps --noconfirm
sudo cp ./*.pkg.tar /home/build/
sudo cp ./*.pkg.tar /var/cache/pacman/pkg/

sudo mkdir -p /package/zfs-utils
sudo cp ./*.pkg.tar /package/zfs-utils/
ls /home/build
sudo pacman -Sy --noconfirm archlinux-keyring
sudo pacman -Syu --noconfirm
popd

# add package to custom repo
repo-add "$CUSTOM_PACMAN_REPO_DB_PATH" /package/zfs-utils/zfs-utils-*.pkg.tar

# sync local package repositores
sudo pacman --noconfirm -Syy

rm -rf zfs-linux
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

sudo mkdir -p /package/zfs-linux
sudo cp ./*.pkg.tar /package/zfs-linux/

popd

# add package to custom repo
repo-add "$CUSTOM_PACMAN_REPO_DB_PATH" /package/zfs-linux/zfs-linux-*.pkg.tar
# sync local package repositores
sudo pacman --noconfirm -Syy
