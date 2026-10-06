#!/bin/bash

# Ensure both parameters are passed
if [ $# -ne 2 ]; then
  echo "Usage: $0 <pkgver> <sha256sum>"
  exit 1
fi

pkgver=$1
sha256sum=$2

# Get the linux version
kernel_version=$(pacman -Q linux | awk '{print $2}')
if [ -z "$kernel_version" ]; then
  echo "Error: kernel version not found"
  exit 1
fi

echo ""
echo ""
echo "Kernel Version: $kernel_version"
echo "pkgver Version: $pkgver"
echo "sha256sum: $sha256sum"
echo ""
echo ""
# Update the PKGBUILD file with the linux version, _zfsver, and sha256sums

# Update the version
sed -i "s|^_zfsver=.*|_zfsver=$pkgver|" PKGBUILD
sed -i "s|^_kernelver=.*|_kernelver=\"$kernel_version\"|" PKGBUILD
sed -i "s|^_kernelver_full=.*|_kernelver_full=\"$kernel_version\"|" PKGBUILD

# Bypass strict kernel version check for newer kernels
sed -i 's/--with-config=kernel/--with-config=kernel --enable-linux-experimental/g' PKGBUILD
# Automatically fetch and update all hashes
updpkgsums


#new_content=$(awk -F"=" -v OFS='=' -v newval="$pkgver" '/^_zfsver/{$2=newval;print;next}1' PKGBUILD)
#echo "$new_content" > PKGBUILD

#new_content=$(awk -F"=" -v OFS='=' -v newval="$kernel_version" '/^_kernelver/{$2=newval;print;next}1' PKGBUILD)
#echo "$new_content" > PKGBUILD

#new_content=$(awk -F"=" -v OFS='=' -v newval="$kernel_version" '/^_kernelver_full/{$2=newval;print;next}1' PKGBUILD)
#echo "$new_content" > PKGBUILD

# NOTE: replaced by updpkgsums
# sed -i -e "s/^sha256sums=.*/sha256sums=('${sha256sum}')/" PKGBUILD

echo "PKGBUILD updated"

echo ""
echo ""
echo ""
cat /home/build/zfs-linux/PKGBUILD
echo ""
echo ""
echo ""