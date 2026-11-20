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
    `SigLevel = Optional TrustAll` makes pacman accept unsigned packages. pacman installs whatever the server
    delivers, with root rights. Use HTTPS, only add a server you control, and turn on signing if you can.

## Signed packages

When the server signs (see [Signing the packages](../guides/signing.md)), import its public key, sign it locally so
pacman trusts it, and require signatures. The Install page of the web UI shows these commands with the fingerprint
filled in:

```bash
curl -fsSL -o zfslocal.asc https://zfs.example.com/zfslocal/x86_64/zfslocal.asc
sudo pacman-key --add zfslocal.asc
sudo pacman-key --lsign-key <fingerprint>
```

Compare the fingerprint with your own record of the key first: the key file comes from the same server as the
packages.

```ini
[zfslocal]
SigLevel = Required
Server = https://zfs.example.com/$repo/$arch
```

pacman then refuses a database or package without a valid signature of that key.

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
