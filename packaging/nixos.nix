# Import alongside the package in environment.systemPackages.
{ lib, pkgs, ... }:
{
  virtualisation.podman.enable = true;
  virtualisation.libvirtd.enable = true;
  environment.systemPackages = [ pkgs.libguestfs ];
  # Add each learner to libvirtd explicitly in the host configuration.
  # Provision per-user storage with scripts/provision_vm_storage.sh.
}
