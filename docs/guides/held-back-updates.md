# Updating while ZFS is held back

`zfs-linux` depends on the exact kernel it was built for. When Arch Linux publishes a kernel that the repository has
no `zfs-linux` for, a full upgrade fails:

```text
:: installing linux (7.3.5.arch1-1) breaks dependency 'linux=7.2.14.arch1-1' required by zfs-linux
```

or, when the installed kernel is older than the one of the repository's `zfs-linux`:

```text
:: unable to satisfy dependency 'linux=7.2.14.arch1-1' required by zfs-linux
```

pacman then changes nothing. That happens for a few minutes after every kernel release, until the build is done, and
for weeks when OpenZFS does not support the new kernel yet. The dashboard shows the card **Updating while ZFS is held
back** while a kernel is behind.

## Update everything except the kernel and ZFS

For the `linux` kernel:

```bash
sudo pacman -Syu --ignore linux,linux-headers,zfs-linux,zfs-linux-headers,zfs-utils
```

For `linux-lts`:

```bash
sudo pacman -Syu --ignore linux-lts,linux-lts-headers,zfs-linux-lts,zfs-linux-lts-headers,zfs-utils
```

The kernel and the ZFS packages stay at the versions you have; everything else is updated. Packages that are not
installed, such as the headers, do not matter in the list.

Ignoring only the kernel is not enough. pacman would still upgrade `zfs-linux` to the newest package of the
repository, which needs a newer kernel than the one you keep. `zfs-utils` is in the list because `zfs-linux` needs the
`zfs-utils` of its own OpenZFS version.

!!! note
    Keeping the kernel while the rest of the system is updated is a partial upgrade, which Arch Linux does not
    support. Holding back only the kernel and its modules usually works, because few packages depend on a kernel
    version, but it is at your own risk. Do the full upgrade as soon as the package is built.

## When the package is built

Run a full upgrade, without `--ignore`:

```bash
sudo pacman -Syu
```

That updates the kernel, its headers and `zfs-linux` in one transaction. Reboot into the new kernel afterwards.

To learn when that is possible without watching the dashboard, alert on the metric `arch_repo_kernel_up_to_date`; see
[Monitoring and alerts](monitoring.md).

## Do not use IgnorePkg

`IgnorePkg` in `/etc/pacman.conf` has the same effect as `--ignore`, but it lasts until you remove it. A forgotten
entry keeps the kernel at an old version without any further error.
