# Failed builds

The server keeps the log of every build in `/repo/logs/<job>.log`, also after the Job or container is gone. Click a
build on the Builds or Analytics page to open it.

## The cause of a failure

The server reads the log of each failed build and puts it in one of three categories. The dashboard shows the
category with the first error line of the log, and the build list shows the category under the status.

| Category | Where the build stopped | Typical causes |
|---|---|---|
| `environment` | Before the kernel module build, while preparing the container. | pacman could not update or install packages, the keyring, a mirror or the network. |
| `kernel` | In the kernel module build, at OpenZFS' `configure` or in the compiler. | OpenZFS does not support the kernel, or the module does not compile against it. |
| `build` | In the kernel module build, for another reason. | A download of the OpenZFS sources, a signature check, `makepkg` itself. |

The border between `environment` and the other two is the line `==> 2/2: Building zfs-linux from AUR...` in the log.

`kernel` failures are expected for a new kernel that OpenZFS does not support yet; see
[New kernels and OpenZFS support](new-kernels.md). The other two are problems of the builder or its surroundings, and
they repeat at every check until someone fixes them.

## Example: a partial upgrade in the build container

In November 2026 every build failed for nine days with:

```text
:: installing openssl (4.1.0-1) breaks dependency 'libcrypto.so=3-64' required by pacman
ERROR: pacman -S --noconfirm --needed base-devel git pacman-contrib openssl curl: exit status 1
```

The builder synced the package database and then installed single packages with `pacman -S`, a partial upgrade. When
Arch Linux moved to openssl 4, pacman upgraded only openssl, which the older pacman and sudo of the
`archlinux:base-devel` image could not use. The builder now runs `pacman -Su` with the packages it needs.

Nobody noticed for nine days because nothing alerted on failed builds. [Monitoring and alerts](monitoring.md) shows
how to get a message after the second failure.

## Retries

A Job retries a failed build twice in the same pod (`backoffLimit: 2`); the log has all attempts one after another.
The first error line is the cause, later attempts often stop differently, for example with `context canceled` when the
Job ends the last attempt.

Independent of that, the server starts a new build at every check (every 6 hours) as long as the kernel has no
packages, except for `kernel` failures, see [When automatic builds start again](new-kernels.md#when-automatic-builds-start-again).

## Retry by hand

After fixing the cause, press **Check Now** on the dashboard: it starts a build for every kernel without packages
right away, instead of at the next check. The Builds page starts a build for one kernel.
