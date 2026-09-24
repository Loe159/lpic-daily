# Topic 101 — Architecture système

Exam: **101-500**

Comprendre matériel, chaîne de boot, init/targets et contrôle propre du cycle de vie système.

## Objectives

### 101.1 — Matériel et périphériques

Weight: **2**  
Recommended practice backend: **podman+host-observation**

**Concepts an agent must teach**
- rôle de /proc, /sys et /dev
- ressources et identification du matériel
- modules du noyau liés au matériel
- bus PCI et USB
- sysfs, udev et D-Bus comme couches de découverte/gestion
- différences entre grandes familles de stockage

**Examinable terms/files/utilities to cover**

`/sys`, `/proc`, `/dev`, `modprobe`, `lsmod`, `lspci`, `lsusb`, `sysfs`, `udev`, `D-Bus`

**Competence evidence**
- interpréter des inventaires matériels
- identifier le module/périphérique pertinent
- expliquer la relation noyau→sysfs/udev→/dev

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

### 101.2 — Chaîne de démarrage Linux

Weight: **3**  
Recommended practice backend: **libvirt-vm**

**Concepts an agent must teach**
- séquence firmware→bootloader→kernel→initramfs→init/systemd
- BIOS versus UEFI
- paramètres noyau au boot
- rôle de initramfs
- SysVinit, systemd et connaissance historique d’Upstart
- diagnostic des événements de boot

**Examinable terms/files/utilities to cover**

`dmesg`, `journalctl`, `BIOS`, `UEFI`, `bootloader`, `kernel`, `initramfs`, `init`, `SysVinit`, `systemd`, `Upstart`

**Competence evidence**
- modifier temporairement un paramètre de boot
- retrouver la cause d’un boot dégradé dans les journaux
- ordonner et expliquer chaque étape de boot

**Modern/legacy note:** Upstart est surtout historique mais reste à reconnaître.

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

### 101.3 — Targets, runlevels, arrêt et redémarrage

Weight: **3**  
Recommended practice backend: **libvirt-vm**

**Concepts an agent must teach**
- runlevels SysV et targets systemd
- mode mono-utilisateur/récupération
- target par défaut
- arrêt/redémarrage propre
- notification aux utilisateurs
- services et dépendances de démarrage
- notion d’événements ACPI

**Examinable terms/files/utilities to cover**

`/etc/inittab`, `shutdown`, `init`, `/etc/init.d`, `telinit`, `systemctl`, `/etc/systemd`, `/usr/lib/systemd`, `wall`, `acpid`

**Competence evidence**
- changer de target et revenir proprement
- diagnostiquer un service bloquant
- programmer/annuler un arrêt et prévenir les sessions

**Modern/legacy note:** SysV runlevels sont legacy mais examinables.

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

