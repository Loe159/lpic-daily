#!/usr/bin/env bash
set -euo pipefail
systemctl set-default multi-user.target
systemctl isolate multi-user.target
systemctl is-active --quiet qemu-guest-agent.service
