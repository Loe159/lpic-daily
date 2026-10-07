#!/usr/bin/env bash
set -euo pipefail
mkdir -p /root/lpic-hardware
lspci -nn > /root/lpic-hardware/pci.txt
lsusb -t > /root/lpic-hardware/usb.txt
lsblk -dn -o NAME,TYPE > /root/lpic-hardware/storage.txt
readlink -f /sys/class/block/vda > /root/lpic-hardware/sys-vda.txt
udevadm info --query=property --name=/dev/vda > /root/lpic-hardware/udev-vda.txt
busctl --no-pager list > /root/lpic-hardware/dbus.txt
modprobe dummy
lsmod > /root/lpic-hardware/modules-loaded.txt
modprobe -r dummy
lsmod > /root/lpic-hardware/modules-final.txt
grep -E '^(Character|Block) devices:' /proc/devices > /root/lpic-hardware/proc-dev.txt
