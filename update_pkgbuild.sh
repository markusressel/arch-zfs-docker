#!/bin/bash

# Ensure both parameters are passed
if [ $# -ne 2 ]; then
  echo "Usage: $0 <pkgver> <sha256sum>"
  exit 1
fi

pkgver=$1
sha256sum=$2

# Get the linux version
kernel_version=$(pacman -Qe | grep "^linux" | awk '{print $2}')
if [ -z "$kernel_version" ]; then
  echo "Error: kernel version not found"
  exit 1
fi

echo ""
echo ""
echo "Kernel Version: $kernel_version"
echo "pkgver Version: $pkgver"
echo "sha256sum: $sha256sum"

# Update the PKGBUILD file with the linux version, _zfsver, and sha256sums

new_content=$(awk -F"=" -v OFS='=' -v newval="$pkgver" '/^_zfsver/{$2=newval;print;next}1' PKGBUILD)
echo "$new_content" > PKGBUILD

new_content=$(awk -F"=" -v OFS='=' -v newval="$kernel_version" '/^_kernelver/{$2=newval;print;next}1' PKGBUILD)
echo "$new_content" > PKGBUILD

new_content=$(awk -F"=" -v OFS='=' -v newval="$kernel_version" '/^_kernelver_full/{$2=newval;print;next}1' PKGBUILD)
echo "$new_content" > PKGBUILD

sed -i -e "s/^sha256sums=.*/sha256sums=('${sha256sum}')/" PKGBUILD

echo "PKGBUILD updated with linux version $kernel_version, _zfsver $pkgver, and sha256sums $sha256sum"
echo ""
echo ""
