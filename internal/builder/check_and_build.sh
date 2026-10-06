#!/bin/bash
set -e

JOB_NAME="${JOB_NAME:-build-$(date +%s)}"
REPO_NAME="${REPO_NAME:-zfslocal}"
VARIANT="${VARIANT:-}"
FORCE_BUILD="${FORCE_BUILD:-false}"
REPO_DIR="/repo/${REPO_NAME}/x86_64"
LOG_DIR="/repo/logs"

mkdir -p "$REPO_DIR" "$LOG_DIR"
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

TARGET_KERNEL=$(pacman -Si "$kernel_pkg" | grep -E '^Version' | awk '{print $3}')
if [ -z "$TARGET_KERNEL" ]; then
  echo "ERROR: Unable to determine latest version for $kernel_pkg"
  exit 1
fi

echo "==> Latest available $kernel_pkg in official repos: $TARGET_KERNEL"

# Check if matching zfs-linux package already exists in repository
EXISTING_PKG=$(find "$REPO_DIR" -maxdepth 1 -name "zfs-${kernel_pkg}-*_${TARGET_KERNEL}*.pkg.tar*" 2>/dev/null | head -n 1)

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
Defaults env_keep += "JOB_NAME VARIANT REPO_NAME FORCE_BUILD KERNEL_VERSION"
EOF_SUDO
chmod 0440 /etc/sudoers.d/build

# Install build requirements
pacman -S --noconfirm --needed base-devel git pacman-contrib openssl "$kernel_pkg" "${kernel_pkg}-headers"

# Configure makepkg for full multi-threading (all CPU cores) and modern compression
sed -i "s|^#*MAKEFLAGS=.*|MAKEFLAGS=\"-j\$(nproc)\"|" /etc/makepkg.conf
sed -i 's/^PKGEXT=.*/PKGEXT=".pkg.tar.zst"/' /etc/makepkg.conf 2>/dev/null || echo 'PKGEXT=".pkg.tar.zst"' >> /etc/makepkg.conf

# Run build as unprivileged user
WORK_DIR="/home/build/work"
rm -rf "$WORK_DIR"
mkdir -p "$WORK_DIR"
chown -R build:build /home/build

# Ensure script directory is executable by build user
chmod +x /scripts/* 2>/dev/null || true

sudo -E -u build bash << 'EOF_BUILD'
set -e
cd /home/build/work

echo "==> Importing OpenZFS maintainer GPG keys..."
gpg --recv-keys 6AD860EED4598027 0AB9E991C6AF658B 2>/dev/null || gpg --keyserver hkps://keyserver.ubuntu.com --recv-keys 6AD860EED4598027 0AB9E991C6AF658B 2>/dev/null || true

echo "==> 1/2: Building zfs-utils from AUR..."
git clone --depth 1 --quiet https://aur.archlinux.org/zfs-utils.git
cd zfs-utils
makepkg --skippgpcheck --install --syncdeps --clean --rmdeps --noconfirm
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

PKGVER=$(grep "^pkgver=" ../zfs-utils/PKGBUILD | awk -F'=' '{print $2}')
SHA256SUM=$(grep "^sha256sums=" ../zfs-utils/PKGBUILD | awk -F"'" '{print $2}')

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
cp -v /home/build/work/zfs-utils/*.pkg.tar* "$REPO_DIR/"
cp -v /home/build/work/zfs-linux*/*.pkg.tar* "$REPO_DIR/"

echo "==> Updating pacman repository database: ${REPO_NAME}.db.tar.zst"
cd "$REPO_DIR"
for pkg in *.pkg.tar*; do
  if [[ "$pkg" != *.sig ]]; then
    repo-add -n -R "${REPO_NAME}.db.tar.zst" "$pkg"
  fi
done

echo "==> Repository successfully updated for $kernel_pkg ($TARGET_KERNEL)!"
ls -la "$REPO_DIR"
