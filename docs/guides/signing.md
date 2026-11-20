# Signing the packages

Without signatures, pacman installs whatever the server delivers, with root rights: whoever can write to the
repository volume, or answer in place of the server, can hand every client a package of their own. With a signing
key, the build jobs sign each package and the database, and clients accept only what carries a valid signature.

Signing needs the Kubernetes runner. The Docker runner builds unsigned packages.

## How the key is used

A build job runs in two containers of one pod:

- `builder` (an init container) builds the packages and copies them into the repository. It runs code from the AUR
  as the build user, which may use sudo, so it is root in its container. It has no access to the key.
- `index` starts when `builder` has finished. It alone mounts the key. It signs the packages that `builder` listed in
  a file on a volume of the pod (not on the repository volume), then rebuilds the database and signs it.

It also publishes the public key next to the database, as `zfslocal.asc` (`<REPO_NAME>.asc`). The Install page of the
web UI then shows its fingerprint and the commands that make a client trust it.

The index step never signs a package that its own build did not hand over. A package on the volume without a valid
signature, unsigned or swapped after the build, is left out of the database with a warning in the job log. A removal
job runs the same index step.

## Set it up

1. Create a key without a passphrase, the job cannot type one. The commands use a throwaway keyring, so the private
   key is never stored on the machine:

    ```bash
    export GNUPGHOME=$(mktemp -d)
    gpg --batch --passphrase '' --quick-gen-key 'zfslocal repository signing key' ed25519 sign never
    fpr=$(gpg --list-keys --with-colons | awk -F: '/^fpr/ {print $10; exit}')
    echo "$fpr"   # note the fingerprint somewhere outside the server
    gpg --armor --export "$fpr" > zfslocal.asc
    gpg --armor --export-secret-keys "$fpr" | kubectl -n arch-repo create secret generic zfslocal-signing-key --from-file=key.asc=/dev/stdin
    rm -rf "$GNUPGHOME"; unset GNUPGHOME
    ```

    `zfslocal.asc` is the public key for the clients. Keep a copy of the private key elsewhere only if you want to;
    a lost key is replaced by a new one and a new fingerprint on the clients.

2. Set `SIGNING_SECRET` in the Deployment to the name of the Secret and restart the server:

    ```yaml
    - name: SIGNING_SECRET
      value: "zfslocal-signing-key"
    ```

    Jobs started from now on sign. A job whose Secret is missing stays pending, it never falls back to unsigned.

3. Sign the packages the database lists today: start one build with **Force build** and **Force rebuild of
   zfs-utils** for each kernel variant you use. Until then the database leaves the unsigned packages out, so wait
   with the clients until these builds are done.

4. Check that the database is signed:

    ```bash
    curl -sI https://zfs.example.com/zfslocal/x86_64/zfslocal.db.sig
    ```

5. On each client, trust the key and require signatures, see [Set up a client](../getting-started/clients.md#signed-packages).

!!! warning
    The Install page, the key file and its fingerprint all come from the server. Someone who can change the volume
    or answer in place of the server can show their own key there. Compare the fingerprint with the one you noted
    when you created the key (`gpg --show-keys zfslocal.asc` shows it) before a client trusts it.

## Key rotation

Create a new key and Secret as in step 1, point `SIGNING_SECRET` at it, rebuild as in step 3, then exchange the key on
the clients. A key that leaked has to be removed from the clients (`pacman-key --delete <fingerprint>`), otherwise
they keep trusting it.
