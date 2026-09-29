# Phase 2 VM image pipeline

This directory defines the trusted, explicit image-supply workflow for the libvirt runner. It is **not** called by LPIC Daily at runtime: missing images fail closed.

`sources.json` pins one x86_64 source for Fedora, Debian and openSUSE. Every source uses HTTPS and an upstream digest embedded directly in the repository. Remote checksum files are not trusted at build time: changing an image and its checksum endpoint together must not silently change the trusted source identity. Floating `latest`, `current` and `daily` paths are rejected by `scripts/validate_vm_image_sources.py`.

Provision the default system-libvirt storage once, then build/install an image as the regular user:

```bash
sudo scripts/provision_vm_storage.sh
python3 scripts/validate_vm_image_sources.py
python3 scripts/build_vm_image.py fedora-44-x86_64-v1 \
  --image-root "/var/lib/libvirt/images/lpic-daily/$(id -u)/images"
```

The default location deliberately lives under libvirt's image tree rather than under `$HOME`: `qemu:///system` must be able to traverse the directories and SELinux/sVirt must be able to label the VM disks. Custom `LPIC_DAILY_VM_IMAGE_DIR` and `LPIC_DAILY_VM_STATE_DIR` paths remain supported, but their DAC/MAC policy is the operator's responsibility. They do not relocate the host-global network-allocation lock: `qemu:///system` shares one host network namespace, so `/var/lib/libvirt/images/lpic-daily/.network-allocation.lock` must still be provisioned once with `scripts/provision_vm_storage.sh`.

Requirements: `qemu-img`, `virt-customize` (libguestfs), enough disk space, and network access for the explicit build. Recipe IDs are executable contracts: the builder selects behavior from the declared recipe ID and rejects recipe/distribution mismatches instead of merely copying a provenance label. The recipe installs the QEMU Guest Agent plus storage utilities, enables a serial getty, makes the boot console serial-capable, scrubs per-machine identity material, validates the qcow2, computes the final SHA-256, installs it read-only, and atomically writes `catalog.json`.

`virtual_size_mb` is also an exact build contract. A source image larger than the declared target is rejected, and the installed image must match the declared target before it can be written to the catalog. The catalog records the exact source URL, recipe identifier, build timestamp and final digest. The runtime re-verifies that final digest before creating a disposable overlay. Rebuilding is repeatable from the pinned source and repository recipe, but the pipeline does not claim byte-for-byte reproducibility while package repositories are not snapshot-pinned.
