# Phase 2 VM image pipeline

This directory defines the trusted, explicit image-supply workflow for the libvirt runner. It is **not** called by LPIC Daily at runtime: missing images fail closed.

`sources.json` pins one x86_64 source for Fedora, Debian and openSUSE. Every source uses HTTPS and mandatory integrity metadata. Fedora and Debian use embedded upstream digests; openSUSE uses its versioned upstream `.sha256` file. Floating `latest`, `current` and `daily` paths are rejected by `scripts/validate_vm_image_sources.py`.

Build/install one image:

```bash
python3 scripts/validate_vm_image_sources.py
python3 scripts/build_vm_image.py fedora-44-x86_64-v1 \
  --image-root "$HOME/.local/share/lpic-daily/vm-images"
```

Requirements: `qemu-img`, `virt-customize` (libguestfs), enough disk space, and network access for the explicit build. The recipe installs the QEMU Guest Agent plus storage utilities, enables a serial getty, makes the boot console serial-capable, scrubs per-machine identity material, validates the qcow2, computes the final SHA-256, installs it read-only, and atomically writes `catalog.json`.

The catalog records the exact source URL, recipe identifier, build timestamp and final digest. The runtime re-verifies that final digest before creating a disposable overlay. Rebuilding is repeatable from the pinned source and repository recipe, but the pipeline does not claim byte-for-byte reproducibility while package repositories are not snapshot-pinned.
