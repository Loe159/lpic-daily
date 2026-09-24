# Topic 102 — Installation Linux et gestion des paquets

Exam: **101-500**

Concevoir stockage/boot, comprendre bibliothèques, administrer paquets Debian/RPM et notions de virtualisation.

## Objectives

### 102.1 — Conception du partitionnement

Weight: **2**  
Recommended practice backend: **libvirt-vm**

**Concepts an agent must teach**
- séparer /, /var, /home et /boot selon usage
- swap et dimensionnement conceptuel
- contraintes /boot et architecture
- EFI System Partition
- partitions et points de montage
- principes de base LVM

**Examinable terms/files/utilities to cover**

`/`, `/var`, `/home`, `/boot`, `ESP`, `swap`, `mount points`, `partitions`, `LVM`

**Competence evidence**
- proposer un schéma adapté à un scénario
- justifier placement/tailles/risques
- reconnaître contraintes BIOS/UEFI

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

### 102.2 — Installation et configuration du bootloader

Weight: **2**  
Recommended practice backend: **libvirt-vm**

**Concepts an agent must teach**
- GRUB Legacy versus GRUB 2
- emplacement/configuration du bootloader
- entrées alternatives et récupération
- MBR et relation avec schéma de boot
- génération de configuration GRUB 2
- interaction au menu/console GRUB

**Examinable terms/files/utilities to cover**

`menu.lst`, `grub.cfg`, `grub.conf`, `grub-install`, `grub-mkconfig`, `MBR`, `GRUB Legacy`, `GRUB 2`

**Competence evidence**
- réparer une VM non amorçable
- ajouter/modifier une entrée de boot
- reconstruire la configuration sans casser la base

**Modern/legacy note:** GRUB Legacy est surtout legacy mais explicitement au programme.

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

### 102.3 — Bibliothèques partagées

Weight: **1**  
Recommended practice backend: **podman**

**Concepts an agent must teach**
- résolution des dépendances dynamiques
- chemins de recherche du chargeur dynamique
- cache ldconfig
- override via variables d’environnement
- diagnostic d’une bibliothèque manquante

**Examinable terms/files/utilities to cover**

`ldd`, `ldconfig`, `/etc/ld.so.conf`, `LD_LIBRARY_PATH`, `shared libraries`

**Competence evidence**
- diagnostiquer un binaire qui ne démarre pas
- expliquer l’ordre de recherche
- mettre à jour correctement le cache

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

### 102.4 — Gestion des paquets Debian

Weight: **3**  
Recommended practice backend: **podman**

**Concepts an agent must teach**
- rôles respectifs dpkg et APT
- installation/mise à jour/suppression
- sources et métadonnées
- dépendances
- inspection d’un paquet et de ses fichiers
- reconfiguration
- identifier quel paquet fournit/possède un fichier

**Examinable terms/files/utilities to cover**

`/etc/apt/sources.list`, `dpkg`, `dpkg-reconfigure`, `apt-get`, `apt-cache`, `apt`

**Competence evidence**
- réparer un état de paquet/dépendance
- retrouver provenance d’un fichier
- installer/supprimer en expliquant couche dpkg vs APT

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

### 102.5 — Gestion RPM, DNF/YUM et Zypper

Weight: **3**  
Recommended practice backend: **podman**

**Concepts an agent must teach**
- format/base RPM
- installation et requêtes rpm
- signature/intégrité
- résolution des dépendances via gestionnaire haut niveau
- repositories YUM/DNF
- workflow Zypper openSUSE
- identifier fichier↔paquet

**Examinable terms/files/utilities to cover**

`rpm`, `rpm2cpio`, `/etc/yum.conf`, `/etc/yum.repos.d`, `yum`, `dnf`, `zypper`

**Competence evidence**
- réaliser mêmes opérations sur Fedora et openSUSE
- inspecter/valider un RPM
- diagnostiquer repository ou dépendance

**Modern/legacy note:** YUM reste examinable; DNF est la pratique moderne Fedora.

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

### 102.6 — Linux invité, virtualisation et cloud

Weight: **1**  
Recommended practice backend: **podman+libvirt-vm**

**Concepts an agent must teach**
- VM versus conteneur applicatif/système
- ressources virtuelles block/network
- images, clones et templates
- identités devant être régénérées lors d’un clonage
- guest drivers
- principes cloud-init/IaaS

**Examinable terms/files/utilities to cover**

`virtual machine`, `Linux container`, `application container`, `guest drivers`, `SSH host keys`, `D-Bus machine-id`, `cloud-init`, `IaaS`

**Competence evidence**
- expliquer différences VM/conteneur
- préparer un clone sans identités dupliquées
- lire une configuration cloud-init simple

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

