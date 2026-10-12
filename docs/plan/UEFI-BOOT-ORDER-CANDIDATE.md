# LPIC-101.2 — UEFI BootOrder recovery

Implemented candidate on the LPIC-101 migration branch. Uses real isolated OVMF NVRAM with two guest EFI boot entries; the setup injects rescue-first order, checks firmware state and retained EFI payloads, and reset must restore a fresh broken state.

Status: **implemented, not accepted**. The Fedora VM image recipe now requires efibootmgr. Rebuild the guest image/catalog before running KVM conformance. Required acceptance: setup red → reference solution green → reset red, alternate solution, no host changes, correct OVMF NVRAM teardown. No real KVM/OVMF execution has been performed for this new lab.

This is 101.2 BIOS-vs-UEFI firmware behavior. It does NOT prove the separate 101.1 requirement to enable or disable integrated hardware in firmware.
