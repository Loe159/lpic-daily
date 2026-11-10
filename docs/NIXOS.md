# Nix / NixOS

Install the pinned, reproducible package:

```sh
nix profile install github:Loe159/lpic-daily
lpic
```

The package embeds the curriculum and includes tmux, notifications, Python,
SSH and VM client tools. The first launch installs user desktop integration
and the daily notification timer. It does not copy the executable out of the
Nix store. `lpic update` directs managed installations to Nix; use
`nix profile upgrade lpic-daily` (or the package name shown by `nix profile list`).
For a declarative installation, add the flake's default package to
`environment.systemPackages` and update the input lock for upgrades.

On NixOS, import `packaging/nixos.nix` and grant the learner libvirt access:

```nix
{
  imports = [ inputs.lpic-daily + "/packaging/nixos.nix" ];
  environment.systemPackages = [ inputs.lpic-daily.packages.${pkgs.stdenv.hostPlatform.system}.default ];
  users.users.YOUR_USER.extraGroups = [ "libvirtd" ];
}
```

This example assumes `inputs` and `pkgs` are module arguments. Rebuild NixOS
and log in again to acquire group membership. Run the narrow provisioning
script once as administrator:

```sh
sudo bash scripts/provision_vm_storage.sh
lpic doctor
```

Podman must come from the NixOS module so rootless helpers and subordinate
UID/GID mappings are configured. VM image creation additionally uses
`virt-customize` from libguestfs; images are downloaded/built only after the
application asks. Full labs require working KVM/libvirt, enough disk space,
and the authored images. Installing the package alone does not validate labs.

Progress lives in `~/.local/state/lpic-daily/progress.sqlite` unless XDG or
LPIC state overrides are set. For migration, stop the application, back up any
existing destination database, then use SQLite's backup API on the source
(including any WAL state) and check `PRAGMA integrity_check`. Copying the
Fedora bootstrap marker is unnecessary: regenerate desktop integration on
NixOS. Keep progress and VM images outside Git and the Nix store.
