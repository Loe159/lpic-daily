#!/usr/bin/env bash
set -euo pipefail
mkdir -p /root/lpic-deb
pkg=$(printf '%s\n' /opt/lpic/packages/nano_*.deb)
{
  cat /etc/apt/sources.list 2>/dev/null || true
  cat /etc/apt/sources.list.d/*.list 2>/dev/null || true
  cat /etc/apt/sources.list.d/*.sources 2>/dev/null || true
} > /root/lpic-deb/sources.txt
apt-cache policy > /root/lpic-deb/policy.txt
apt-cache depends nano > /root/lpic-deb/dependencies.txt
apt-get remove -y nano
apt-get install -y "$pkg"
dpkg -r nano
dpkg -i "$pkg"
dpkg-deb -c "$pkg" > /root/lpic-deb/package-content.txt
dpkg -L nano > /root/lpic-deb/installed-files.txt
dpkg -S /usr/bin/bash > /root/lpic-deb/owner.txt
apt-get -s upgrade > /root/lpic-deb/upgrade-sim.txt
printf 'tzdata tzdata/Areas select Etc\ntzdata tzdata/Zones/Etc select UTC\n' | debconf-set-selections
DEBIAN_FRONTEND=noninteractive dpkg-reconfigure -f noninteractive tzdata >/root/lpic-deb/reconfigure.txt 2>&1
readlink -f /etc/localtime > /root/lpic-deb/timezone-target.txt
