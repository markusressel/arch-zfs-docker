#!/bin/bash

# makepkg cannot (and should not) be run as root, so we create a custom user named "build"
useradd --create-home build

# install the requested linux kernel version (and other required tools)
pacman -Sy --noconfirm base-devel git "linux$VARIANT" "linux$VARIANT-headers"
# do a full system upgrade
pacman -Syu --noconfirm

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

[custom]
SigLevel = Never
Server = file:///home/build

[core]
Include = /etc/pacman.d/mirrorlist

[extra]
Include = /etc/pacman.d/mirrorlist
EOL

# Disable compression on local build and enable SMP
cat >> /etc/makepkg.conf << EOL
PKGEXT='.pkg.tar'
MAKEFLAGS="-j17"
EOL

echo "Pacman Configuration:"
cat /etc/pacman.conf

echo "Makepkg COnfig:"
cat /etc/makepkg.conf
