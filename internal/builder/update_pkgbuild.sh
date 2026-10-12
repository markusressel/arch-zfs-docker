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

# OpenZFS refuses kernels newer than its META Linux-Maximum. Only an explicit request skips that check:
# such modules are untested by OpenZFS, and pacman would install them on every client.
if [ "${ALLOW_UNSUPPORTED_KERNEL}" = "true" ]; then
  echo "==> WARNING: building even if OpenZFS does not support this kernel (--enable-linux-experimental)"
  sed -i 's/--with-config=kernel/--with-config=kernel --enable-linux-experimental/g' PKGBUILD
fi

# The AUR zfs-linux PKGBUILD pins only the tarball's checksum, which has to be recomputed when the
# OpenZFS version changes. Verify the release signature instead, like the zfs-utils PKGBUILD does.
cat >> PKGBUILD << 'EOF_SIG'

source+=("${source[0]}.asc")
validpgpkeys=('4F3BA9AB6D1F8D683DC2DFB56AD860EED4598027'  # Tony Hutter (GPG key for signing ZFS releases)
              'C33DF142657ED1F7C328A2960AB9E991C6AF658B') # Brian Behlendorf
EOF_SIG

# Recompute the checksums for the new version; makepkg then checks the tarball against the signature.
updpkgsums
echo "==> PKGBUILD updated successfully."
