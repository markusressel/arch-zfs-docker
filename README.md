# introduction

i needed a way to build my own zfs-linux-lts and zfs-utils, so came up with this solution

[check out the blog article behind it](https://jbrio.net/posts/5-ways-archlinux-zfs/)

## purpose

this docker will build

- zfs-utils
- zfs-linux-lts

against the latest available linux-lts kernel

## usage

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

- clone/download this repo to your system, then


```bash
REPOSITORY_PATH=/home/markus/tmp/zfs VARIANT="lts" make packages
```

will create zfs-linux-lts & zfs-utils folders below ~/tmp/zfs


TODO: maybe use /var/cache/pacman/pkg/ instead of "~/tmp/zfs" ?
TODO: add parameter to switch between LTS and non-LTS?



### Cleanup

Generally this should not be necessary, but in case you want to start fresh:

```bash
make clean
```

or

```bash
docker rmi $(docker images --filter=reference="archbuild" -q)
```
