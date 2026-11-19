# New kernels and OpenZFS support

Every OpenZFS release declares the newest Linux version it supports, in the line `Linux-Maximum` of its `META` file.
For a newer kernel, its `configure` step refuses to build:

```text
configure: error:
	*** Cannot build against kernel version 7.3.5-arch1-1.
	*** The maximum supported kernel version is 7.2.
```

Arch Linux often releases a new kernel before OpenZFS supports it, so the repository falls behind until a new OpenZFS
release reaches the AUR. The server tracks this for each kernel that is not up to date.

## The states

The dashboard shows one of these for a kernel without packages, and the metric `arch_repo_kernel_compat` has the
same names:

| State | Meaning | What the server does |
|---|---|---|
| `ok` | The kernel is within `Linux-Maximum` of the OpenZFS version the next build packages. | Builds it. |
| `predicted_unsupported` | The kernel is newer than `Linux-Maximum`. | Builds it anyway; `Linux-Maximum` is what OpenZFS tested, and `configure` decides. |
| `incompatible` | A build of this kernel with this OpenZFS version failed because of the kernel. | Pauses automatic builds for the pair. |
| `unknown` | The AUR or GitHub could not be reached. | Builds it. |

The OpenZFS version comes from the `zfs-utils` package in the AUR, and `Linux-Maximum` from the `META` file of that
release on GitHub.

## When automatic builds start again

An `incompatible` verdict belongs to one kernel version and one OpenZFS version. Automatic builds start again when:

- a new OpenZFS release is packaged in the AUR,
- Arch Linux publishes another kernel version, or
- 7 days have passed since the failure.

The verdicts are stored in `/repo/compat.json`.

## Related OpenZFS issues

For `predicted_unsupported` and `incompatible` the dashboard lists up to five OpenZFS issues whose title names the
kernel version as a Linux or kernel version, for example `Kernel 7.3-rc1 compatibility with OpenZFS 2.4.4`, and links
to a search on GitHub. Support for a new kernel usually lands in the OpenZFS repository first and in a release later.

## Build an unsupported kernel anyway

The build form on the Builds page has the checkbox **Build even if OpenZFS does not support this kernel yet**. It
passes `--enable-linux-experimental` to `configure`, which skips the version check. The module may build, and it may
not work: OpenZFS has not tested it with that kernel.

!!! warning
    A package built this way is installed by every client on its next `pacman -Syu`. Use it only on a repository whose
    clients can deal with a broken ZFS module, for example a test machine that does not boot from ZFS.

The server never sets this option for automatic builds.

## Older kernels and other OpenZFS versions

The build form also builds a specific kernel version (from the Arch Linux Archive) or a specific OpenZFS release. The
pacman database lists the newest package per name, so an older build stays in the repository directory without
replacing what clients install.
