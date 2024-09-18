# introduction

i needed a way to build my own zfs-linux-lts and zfs-utils, so came up with this solution

[check out the blog article behind it](https://jbrio.net/posts/5-ways-archlinux-zfs/)

## purpose

this docker will build

- zfs-utils
- zfs-linux-lts

against the latest available linux-lts kernel

## usage

- clone/download this repo to your system, then

```bash
# make sure to start with a fresh image
docker rmi $(docker images --filter=reference="archbuild" -q)
docker buildx build --tag archbuild --build-arg VARIANT="" .
mkdir -p ~/tmp/zfs
docker run -i -t --rm -v ~/tmp/zfs:/package archbuild
```

will create zfs-linux-lts & zfs-utils folders below ~/tmp/zfs


TODO: maybe use /var/cache/pacman/pkg/ instead of "~/tmp/zfs" ?
TODO: add parameter to switch between LTS and non-LTS?
