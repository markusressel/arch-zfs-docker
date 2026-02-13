#!/bin/bash

# makepkg cannot (and should not) be run as root, so we create a custom user named "build"
useradd --create-home build

echo "Variant: $VARIANT"
if [[ "$VARIANT" == "lts" ]]; then
    VARIANT="-lts"
fi

# install the requested linux kernel version (and other required tools)
pacman -Sy --noconfirm base-devel git pacman-contrib "linux$VARIANT" "linux$VARIANT-headers"
# do a full system upgrade
pacman -Syu --noconfirm

# setup git (to prevent warning about default branch name)
sudo git config --system init.defaultbranch main

# Allow the "build" user to run stuff as root (to install dependencies)
echo "build ALL=(ALL) NOPASSWD: ALL" > /etc/sudoers.d/build

# Add Custom Repo Entry to /etc/pacman.conf
cat > /etc/pacman.conf << EOL
[options]
HoldPkg     = pacman glibc
Architecture = auto
Color
CheckSpace
ParallelDownloads = 32
# This allows us to work even if the keyring is momentarily out of sync
SigLevel = Never 

[core]
Include = /etc/pacman.d/mirrorlist

[extra]
Include = /etc/pacman.d/mirrorlist

# Move [custom] to the bottom so it's checked last
[custom]
SigLevel = Optional TrustAll
Server = file:///home/build/repo
EOL

# Disable compression on local build and enable SMP
cat >> /etc/makepkg.conf << EOL
PKGEXT='.pkg.tar'
MAKEFLAGS="-j17"
OPTIONS=(strip docs !libtool !staticlibs emptydirs zipman purge !debug !lto)
EOL
