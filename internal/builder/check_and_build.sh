#!/bin/bash
set -e

JOB_NAME="${JOB_NAME:-build-$(date +%s)}"
REPO_NAME="${REPO_NAME:-zfslocal}"
VARIANT="${VARIANT:-}"
FORCE_BUILD="${FORCE_BUILD:-false}"
FORCE_REBUILD_UTILS="${FORCE_REBUILD_UTILS:-false}"
KERNEL_VERSION="${KERNEL_VERSION:-}"
ZFS_VERSION="${ZFS_VERSION:-}"
REPO_DIR="/repo/${REPO_NAME}/x86_64"
LOG_DIR="/repo/logs"
CACHE_DIR="/repo/cache"

export JOB_NAME REPO_NAME VARIANT FORCE_BUILD FORCE_REBUILD_UTILS KERNEL_VERSION ZFS_VERSION REPO_DIR LOG_DIR CACHE_DIR

mkdir -p "$REPO_DIR" "$LOG_DIR" "$CACHE_DIR"
exec > >(tee -a "${LOG_DIR}/${JOB_NAME}.log") 2>&1

kernel_pkg="linux"
if [ -n "$VARIANT" ]; then
  kernel_pkg="linux-${VARIANT}"
fi

echo "==> Initializing pacman keyring..."
pacman-key --init || true
pacman-key --populate archlinux || true

echo "==> Refreshing Arch pacman database..."
pacman -Sy --noconfirm archlinux-keyring
pacman -Sy --noconfirm

ARCHIVE_URL="https://archive.archlinux.org/packages/l"
PINNED_KERNEL_PKGS=()

if [ -n "$KERNEL_VERSION" ]; then
  # Specific kernel requested: install exactly that version from the Arch Linux Archive.
  TARGET_KERNEL="${KERNEL_VERSION/-arch/.arch}"
  echo "==> Requested $kernel_pkg version: $TARGET_KERNEL"
  PINNED_KERNEL_PKGS=(
    "${ARCHIVE_URL}/${kernel_pkg}/${kernel_pkg}-${TARGET_KERNEL}-x86_64.pkg.tar.zst"
    "${ARCHIVE_URL}/${kernel_pkg}-headers/${kernel_pkg}-headers-${TARGET_KERNEL}-x86_64.pkg.tar.zst"
  )
  for url in "${PINNED_KERNEL_PKGS[@]}"; do
    if ! curl -fsIL -o /dev/null "$url"; then
      echo "ERROR: $url not found. Kernel version $TARGET_KERNEL is not available in the Arch Linux Archive."
      exit 1
    fi
  done
else
  TARGET_KERNEL=$(pacman -Si "$kernel_pkg" | grep -E '^Version' | awk '{print $3}')
  if [ -z "$TARGET_KERNEL" ]; then
    echo "ERROR: Unable to determine latest version for $kernel_pkg"
    exit 1
  fi
  echo "==> Latest available $kernel_pkg in official repos: $TARGET_KERNEL"
fi

# Package file names use dots only (e.g. zfs-linux-2.4.4_7.2.8.arch1.2-1-x86_64.pkg.tar.zst)
TARGET_KERNEL_DOTTED="${TARGET_KERNEL//-/.}"

# Check if matching zfs-linux package already exists in repository
EXISTING_PKG=$(find "$REPO_DIR" -maxdepth 1 -name "zfs-${kernel_pkg}-${ZFS_VERSION:-[0-9]*}_${TARGET_KERNEL_DOTTED}-*.pkg.tar*" 2>/dev/null | head -n 1)

if [ -n "$EXISTING_PKG" ] && [ "$FORCE_BUILD" != "true" ]; then
  echo "==> Package for $kernel_pkg ($TARGET_KERNEL) already exists in $REPO_DIR:"
  echo "    $EXISTING_PKG"
  echo "==> Repository is up to date. Exiting cleanly."
  exit 0
fi

echo "==> New kernel detected or FORCE_BUILD=true. Preparing build environment..."

# Configure build user
if ! id -u build &>/dev/null; then
  useradd --create-home --shell /bin/bash build
fi
cat << 'EOF_SUDO' > /etc/sudoers.d/build
build ALL=(ALL) NOPASSWD: ALL
Defaults env_keep += "JOB_NAME VARIANT REPO_NAME FORCE_BUILD FORCE_REBUILD_UTILS KERNEL_VERSION ZFS_VERSION REPO_DIR CACHE_DIR"
EOF_SUDO
chmod 0440 /etc/sudoers.d/build

# Configure package and source caching on persistent volume
CACHE_DIR="/repo/cache"
mkdir -p "$CACHE_DIR/pacman" "$CACHE_DIR/sources"
sed -i "s|^#*CacheDir.*|CacheDir = $CACHE_DIR/pacman /var/cache/pacman/pkg|" /etc/pacman.conf

# Install build requirements
pacman -S --noconfirm --needed base-devel git pacman-contrib openssl curl
if [ ${#PINNED_KERNEL_PKGS[@]} -gt 0 ]; then
  pacman -U --noconfirm "${PINNED_KERNEL_PKGS[@]}"
else
  pacman -S --noconfirm --needed "$kernel_pkg" "${kernel_pkg}-headers"
fi

# Configure makepkg: all CPU cores, persistent source caching, fast compression, no debug package
sed -i "s|^#*MAKEFLAGS=.*|MAKEFLAGS=\"-j\$(nproc)\"|" /etc/makepkg.conf
sed -i "s|^#*SRCDEST=.*|SRCDEST=$CACHE_DIR/sources|" /etc/makepkg.conf
sed -i 's/^PKGEXT=.*/PKGEXT=".pkg.tar.zst"/' /etc/makepkg.conf 2>/dev/null || echo 'PKGEXT=".pkg.tar.zst"' >> /etc/makepkg.conf
sed -i 's|^#*COMPRESSZST=.*|COMPRESSZST=(zstd -c -T0 -3 -)|' /etc/makepkg.conf
sed -i 's/OPTIONS=(.*)/OPTIONS=(strip docs !libtool !staticlibs emptydirs zipman purge !debug lto)/' /etc/makepkg.conf

# Run build as unprivileged user
WORK_DIR="/home/build/work"
rm -rf "$WORK_DIR"
mkdir -p "$WORK_DIR"
chown -R build:build /home/build "$CACHE_DIR"

# Ensure script directory is executable by build user
chmod +x /scripts/* 2>/dev/null || true

sudo -E -u build bash << 'EOF_BUILD'
set -e
cd /home/build/work

echo "==> Importing OpenZFS maintainer GPG keys..."
gpg --recv-keys 6AD860EED4598027 0AB9E991C6AF658B 2>/dev/null || gpg --keyserver hkps://keyserver.ubuntu.com --recv-keys 6AD860EED4598027 0AB9E991C6AF658B 2>/dev/null || true

echo "==> 1/2: Checking zfs-utils from AUR..."
git clone --depth 1 --quiet https://aur.archlinux.org/zfs-utils.git
cd zfs-utils

if [ -n "$ZFS_VERSION" ]; then
  echo "==> Requested OpenZFS version: $ZFS_VERSION (overriding the AUR default)"
  sed -i "s|^pkgver=.*|pkgver=$ZFS_VERSION|; s|^pkgrel=.*|pkgrel=1|" PKGBUILD
  updpkgsums
fi

PKGVER=$(grep "^pkgver=" PKGBUILD | awk -F'=' '{print $2}')
SHA256SUM=$(grep "^sha256sums=" PKGBUILD | awk -F"'" '{print $2}')

# Check if matching zfs-utils package already exists in repository
EXISTING_UTILS=$(find "$REPO_DIR" -maxdepth 1 -name "zfs-utils-${PKGVER}-*.pkg.tar*" 2>/dev/null | grep -v 'debug' | head -n 1 || true)

if [ -n "$EXISTING_UTILS" ] && [ "$FORCE_REBUILD_UTILS" != "true" ]; then
  echo "==> Existing zfs-utils package found: $EXISTING_UTILS"
  echo "==> Skipping zfs-utils compilation and installing from repository cache..."
  sudo pacman -U --noconfirm "$EXISTING_UTILS"
else
  echo "==> Building zfs-utils from AUR..."
  makepkg --skippgpcheck --install --syncdeps --clean --rmdeps --noconfirm
fi
cd ..

echo "==> 2/2: Building zfs-linux from AUR..."
VARIANT_ARG="${VARIANT}"
if [ -n "$VARIANT_ARG" ]; then
  VARIANT_SUFFIX="-${VARIANT_ARG}"
else
  VARIANT_SUFFIX=""
fi

git clone --depth 1 --quiet "https://aur.archlinux.org/zfs-linux${VARIANT_SUFFIX}.git" "zfs-linux${VARIANT_SUFFIX}"
cd "zfs-linux${VARIANT_SUFFIX}"

/scripts/update_pkgbuild.sh "$PKGVER" "$SHA256SUM"

# Generate self-signed module signing key
openssl req -new -nodes -utf8 -sha512 -days 36500 -batch -x509 \
  -subj "/C=US/ST=State/L=City/O=Organization/OU=Unit/CN=Regenerated-ZFS-Key" \
  -outform DER -out signing_key.x509 -keyout signing_key.pem

makepkg --skippgpcheck --syncdeps --clean --rmdeps --noconfirm
cd ..
echo "==> Build completed successfully."
EOF_BUILD

echo "==> Publishing packages to repository directory: $REPO_DIR"
shopt -s nullglob
for f in /home/build/work/zfs-utils/*.pkg.tar.zst /home/build/work/zfs-linux*/*.pkg.tar.zst; do
  cp -v "$f" "$REPO_DIR/"
done

echo "==> Updating pacman repository database: ${REPO_NAME}.db.tar.zst"
/scripts/update_repo_db.sh "$REPO_DIR" "$REPO_NAME"

echo "==> Repository successfully updated for $kernel_pkg ($TARGET_KERNEL)!"
ls -la "$REPO_DIR"
