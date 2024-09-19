# arch-zfs-docker

Helper scripts to build zfs-linux(-lts) and zfs-utils locally.

## Usage

### Setup a Local Custom Repository

```bash
REPOSITORY_PATH=/home/markus/.custom/zfs

mkdir -p "$REPOSITORY_PATH"

# add entry to pacman.conf for custom repository
cat >> /etc/pacman.conf << EOL

[zfslocal]
SigLevel = Optional TrustAll
Server = file://$REPOSITORY_PATH

EOL
```

### Populate the Repository

Clone this repo to your system and run the following command:


```bash
REPOSITORY_NAME="zfslocal" REPOSITORY_PATH=/home/markus/.custom/zfs ./populate-package-repository.sh

# for LTS version
REPOSITORY_NAME="zfslocal" REPOSITORY_PATH=/home/markus/.custom/zfs VARIANT="lts" ./populate-package-repository.sh
```

This will ensure the repository path exists, build packages for both `zfs-utils` as well
 as `zfs-linux` (or `zfs-linux-lts`) and add them to the given repository.

To build the packages, the latest available `linux` (or `linux-lts`) kernel is used.  
The resulting package archives will be placed into the path given by `$REPOSITORY_PATH`.

If you have correctly setup the local custom repository for pacman like mentioned above, the newly built packages
will be automatically picked up by pacman when upgrading the system.

### Cleanup

Generally this should not be necessary, but in case you want to start fresh:

```bash
make clean
```

or

```bash
docker rmi $(docker images --filter=reference="archbuild" -q)
```

# Attributions

Thx to @jbrodriguez for the upstream version of this repository.
Without his/her efforts I would not have created this.
