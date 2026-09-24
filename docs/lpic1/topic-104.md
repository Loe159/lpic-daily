# Topic 104 — Périphériques, filesystems et FHS

Exam: **101-500**

Administrer partitions/filesystems, montages, permissions, liens et organisation standard du système.

## Objectives

### 104.1 — Partitions et création de systèmes de fichiers

Weight: **2**  
Recommended practice backend: **libvirt-vm**

**Concepts an agent must teach**
- MBR versus GPT
- création/modification de partitions
- ext2/ext3/ext4
- XFS
- VFAT/exFAT
- swap
- principes Btrfs multi-device/compression/subvolumes
- création de filesystem

**Examinable terms/files/utilities to cover**

`fdisk`, `gdisk`, `parted`, `mkfs`, `mkswap`, `ext2`, `ext3`, `ext4`, `XFS`, `VFAT`, `exFAT`, `Btrfs`

**Competence evidence**
- partitionner disques virtuels
- créer fs/swap demandés
- choisir outil/type approprié sans toucher hôte

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

### 104.2 — Intégrité et maintenance des filesystems

Weight: **2**  
Recommended practice backend: **libvirt-vm**

**Concepts an agent must teach**
- occupation blocs/inodes
- vérification ext
- réparation
- tuning ext
- outils XFS
- différence outils online/offline selon fs
- symptômes de corruption/espace épuisé

**Examinable terms/files/utilities to cover**

`du`, `df`, `fsck`, `e2fsck`, `mke2fs`, `tune2fs`, `xfs_repair`, `xfs_fsr`, `xfs_db`

**Competence evidence**
- diagnostiquer inode vs blocs
- réparer copie de fs volontairement endommagée
- sélectionner outil spécifique au fs

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

### 104.3 — Montage et démontage

Weight: **3**  
Recommended practice backend: **libvirt-vm**

**Concepts an agent must teach**
- montage manuel
- fstab persistant
- UUID/labels
- options de montage
- démontage et fichiers occupés
- unités mount systemd
- périphériques amovibles

**Examinable terms/files/utilities to cover**

`/etc/fstab`, `/media`, `mount`, `umount`, `blkid`, `lsblk`, `UUID`, `labels`, `systemd mount units`

**Competence evidence**
- créer montage persistant qui survit au reboot
- corriger fstab cassé
- retrouver identité d’un volume

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

### 104.5 — Permissions et ownership

Weight: **3**  
Recommended practice backend: **podman**

**Concepts an agent must teach**
- permissions rwx fichier/dossier
- notation symbolique/octale
- propriétaire/groupe
- umask
- SUID
- SGID
- sticky bit
- effets spéciaux sur répertoires

**Examinable terms/files/utilities to cover**

`chmod`, `umask`, `chown`, `chgrp`, `SUID`, `SGID`, `sticky bit`

**Competence evidence**
- concevoir permissions minimales
- diagnostiquer accès refusé
- mettre en place dossier collaboratif correctement

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

### 104.6 — Liens symboliques et liens physiques

Weight: **2**  
Recommended practice backend: **podman**

**Concepts an agent must teach**
- inode et hard link
- symlink et cible
- limites filesystem/directory des hard links
- liens cassés
- différence copie/lien
- usages administratifs

**Examinable terms/files/utilities to cover**

`ln`, `ls`, `inode`, `hard link`, `symbolic link`

**Competence evidence**
- prédire effets suppression/renommage
- créer bons types de liens
- identifier liens/inodes

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

### 104.7 — FHS et localisation de fichiers

Weight: **2**  
Recommended practice backend: **podman**

**Concepts an agent must teach**
- rôle des principaux répertoires FHS
- recherche temps réel vs base locate
- mise à jour base locate
- localiser binaire/source/man
- résolution de commande shell

**Examinable terms/files/utilities to cover**

`find`, `locate`, `updatedb`, `whereis`, `which`, `type`, `/etc/updatedb.conf`, `FHS`

**Competence evidence**
- choisir bon outil de localisation
- expliquer où devrait vivre un fichier selon sa fonction
- diagnostiquer commande trouvée inattendue

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

