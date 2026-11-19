# Set up a client

A client is an Arch Linux machine that installs `zfs-linux` and `zfs-utils` from the repository. The Install page of
the web UI shows the same steps with the address of your server filled in.

## Add the repository

Add this section to `/etc/pacman.conf`, above `[core]`, so pacman prefers these packages over other repositories:

```ini
[zfslocal]
SigLevel = Optional TrustAll
Server = https://zfs.example.com/$repo/$arch
```

`zfslocal` is the default of `REPO_NAME`; use the name your server is configured with.

!!! warning
    The packages are not signed, and `SigLevel = Optional TrustAll` makes pacman accept them. pacman installs
    whatever the server delivers, with root rights. Use HTTPS, and only add a server you control.

## Install ZFS

For the `linux` kernel:

```bash
sudo pacman -Syu zfs-linux zfs-utils
```

For `linux-lts`:

```bash
sudo pacman -Syu zfs-linux-lts zfs-utils
```

`-Syu` updates the whole system together with the new packages. `pacman -Sy` followed by installing single packages
is a partial upgrade, which Arch Linux does not support.

## Updates

From now on a normal `sudo pacman -Syu` updates the kernel and `zfs-linux` together. That works as long as the
repository has a `zfs-linux` for the newest Arch kernel. The server notices a new kernel at its next check (every
6 hours by default), and the build takes a few minutes more.

In between, or for longer when OpenZFS does not support the new kernel yet, `pacman -Syu` stops with an error about
the `linux=...` dependency of `zfs-linux` and changes nothing. [Updating while ZFS is held back](../guides/held-back-updates.md)
explains what to do then.
