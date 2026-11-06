# Package-test VMs

One Packer/Tart recipe creates a fresh Ubuntu 22.04 ARM64 guest with Homebrew.
It uses 2 CPUs, 4 GiB RAM and a 30 GB disk. The base image and Homebrew installer
are pinned; apt packages and the installed Homebrew revision are recorded in
the build log and the guest's `~/patronus-vm-tools.txt`.

```text
+-----------------+  Packer/Tart  +-------------------+  shell script  +----------------+
| Pinned Ubuntu   | ============> | New package VM    | =============> | Native checks  |
+-----------------+              +-------------------+                +----------------+
```

Use Tart 2.41.1, Packer 1.16.2 and the Tart plugin 1.21.0. Keep normal host,
network, SSH and package-manager permission checks enabled. Tart supports Linux
and macOS guests; this recipe does not create a Windows VM.

From the repository root, choose a new VM name that is absent from `tart list`:

```sh
export TART_NO_AUTO_PRUNE=1
packer init tools/vms/ubuntu.pkr.hcl
packer fmt -check tools/vms/ubuntu.pkr.hcl
packer validate -var vm_name=patronus-package-example tools/vms/ubuntu.pkr.hcl
tart list --format json
packer build -on-error=abort -var vm_name=patronus-package-example tools/vms/ubuntu.pkr.hcl
```

The default check reports guest and Homebrew versions. To test packages, pass
`-var check_script=/absolute/path/check.sh` to `packer build`. Packer uploads
that script through SSH and runs it as the ordinary guest `admin` account.
The script can contain a synthetic package fixture or run a separately approved
release test. A successful setup probe is not package-install acceptance.

Builds retain their VM. Use a new name for every clean run; never use an
overwrite/force option, automatic pruning, host directory sharing or credential
forwarding. Stop an owned guest after a failed run and retain its logs. Successful
Packer builds stop the guest normally. The stock `admin/admin` login is public
image data and grants no repository access.
