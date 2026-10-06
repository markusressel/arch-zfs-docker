#!/bin/bash

PACKAGE_DIR="/packages"

#############
# zfs-linux #
#############

git clone --depth 1 --quiet -b bump-zfs-to-2.2.6 https://github.com/prikhi/archzfs.git
pushd archzfs || exit
  sudo ./build.sh std
  cp ./*.pkg.tar "$PACKAGE_DIR"
popd || exit