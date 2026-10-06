#!/bin/bash
set -e
if [ $# -ne 2 ]; then
  echo "Usage: $0 <pkgver> <sha256sum>"
  exit 1
fi

pkgver="$1"
sha256sum="$2"
variant="${VARIANT}"
if [ -n "$variant" ] && [ "$variant" != "lts" ]; then
  variant="-$variant"
elif [ "$variant" = "lts" ]; then
  variant="-lts"
fi

kernel_pkg="linux${variant}"
kernel_version=$(pacman -Q "$kernel_pkg" | awk '{print $2}')
if [ -z "$kernel_version" ]; then
  echo "Error: kernel version for $kernel_pkg not found"
  exit 1
fi

echo "==> Updating PKGBUILD:"
echo "    Kernel Package: $kernel_pkg"
echo "    Kernel Version: $kernel_version"
echo "    ZFS pkgver:     $pkgver"

sed -i "s|^_zfsver=.*|_zfsver=$pkgver|" PKGBUILD
sed -i "s|^_kernelver=.*|_kernelver=\"$kernel_version\"|" PKGBUILD
sed -i "s|^_kernelver_full=.*|_kernelver_full=\"$kernel_version\"|" PKGBUILD
if [ "$variant" = "-lts" ]; then
  sed -i "s|^_extramodules=.*|_extramodules=\"${kernel_version}-lts\"|" PKGBUILD
fi

# Bypass strict kernel version check for newer kernels
sed -i 's/--with-config=kernel/--with-config=kernel --enable-linux-experimental/g' PKGBUILD

# Automatically fetch and update all hashes
updpkgsums
echo "==> PKGBUILD updated successfully."
