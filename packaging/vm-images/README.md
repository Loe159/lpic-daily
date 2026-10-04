# Trusted VM image pipeline

This directory defines the trusted image-supply contract used by the libvirt runner.

LPIC Daily can now invoke this pipeline during `lpic install` or lazily before the first VM lab, but only after explicit user confirmation for privileged/network-heavy steps. The same scripts remain usable directly by maintainers.

`sources.json` pins one x86_64 source for Fedora, Debian and openSUSE. Every source uses HTTPS plus an integrity digest embedded in the repository. Floating `latest`, `current` and `daily` paths are rejected.

Manual workflow:

```bash
sudo scripts/provision_vm_storage.sh
python3 scripts/validate_vm_image_sources.py
python3 scripts/build_vm_image.py fedora-44-x86_64-v2 \
  --image-root "/var/lib/libvirt/images/lpic-daily/$(id -u)/images"
```

The default storage remains below libvirt's image tree rather than `$HOME` so `qemu:///system` and SELinux/sVirt can access and label managed disks. Custom VM image/state paths are supported only when the operator supplies equivalent DAC/MAC policy. The host-global network allocation lock remains shared because `qemu:///system` has one host network namespace.

Requirements: `qemu-img`, `virt-customize`, sufficient disk space and explicit network access for source/package retrieval.

Recipe IDs are executable contracts. The builder:
- verifies the pinned upstream source digest;
- installs the declared packages;
- configures QEMU Guest Agent and serial console support;
- applies the project guest-exec trampoline;
- scrubs per-machine identity;
- validates the QCOW2;
- enforces the declared virtual size;
- computes the final SHA-256;
- installs the image read-only;
- atomically updates `catalog.json`.

Runtime re-verifies the final digest before creating a disposable overlay. Rebuilding is repeatable from the pinned source and recipe, but byte-for-byte reproducibility is not claimed while package repositories are not snapshot-pinned.
