# Topic 107 — Tâches administratives

Exam: **102-500**

Gérer identités, planification, locales, encodages et fuseaux horaires.

## Objectives

### 107.1 — Utilisateurs et groupes

Weight: **5**  
Recommended practice backend: **podman**

**Concepts an agent must teach**
- bases passwd/shadow/group
- création/modification/suppression comptes
- verrouillage et expiration
- groupes primaires/secondaires
- comptes système/à usage limité
- squelettes
- lookup via NSS/getent

**Examinable terms/files/utilities to cover**

`/etc/passwd`, `/etc/shadow`, `/etc/group`, `/etc/skel`, `chage`, `getent`, `groupadd`, `groupdel`, `groupmod`, `passwd`, `useradd`, `userdel`, `usermod`

**Competence evidence**
- administrer cycle de vie utilisateurs/groupes
- diagnostiquer incohérences de droits/identité
- mettre politiques expiration/verrouillage

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

### 107.2 — Planification des tâches

Weight: **4**  
Recommended practice backend: **podman+libvirt-vm**

**Concepts an agent must teach**
- crontab utilisateur/système
- syntaxe calendrier cron
- at tâches ponctuelles
- allow/deny
- spool concept
- systemd timers et transient timers
- choix cron vs timer

**Examinable terms/files/utilities to cover**

`cron`, `/etc/cron.*`, `at.allow`, `at.deny`, `crontab`, `cron.allow`, `cron.deny`, `at`, `atq`, `atrm`, `systemctl`, `systemd-run`, `systemd timers`

**Competence evidence**
- planifier job récurrent et ponctuel
- diagnostiquer job non exécuté
- convertir besoin simple en timer systemd

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

### 107.3 — Locales, encodages et fuseaux horaires

Weight: **3**  
Recommended practice backend: **podman+libvirt-vm**

**Concepts an agent must teach**
- locale et catégories LC_*
- LANG/LC_ALL priorité
- timezone système/utilisateur
- UTC vs heure locale
- UTF-8/Unicode/ASCII/ISO-8859
- conversion encodage
- configuration timezone avec timedatectl/tzselect

**Examinable terms/files/utilities to cover**

`/etc/timezone`, `/etc/localtime`, `/usr/share/zoneinfo`, `LC_*`, `LC_ALL`, `LANG`, `TZ`, `locale`, `tzselect`, `timedatectl`, `date`, `iconv`, `UTF-8`, `ISO-8859`, `ASCII`, `Unicode`

**Competence evidence**
- corriger tri/affichage influencé par locale
- convertir fichier d’encodage
- changer timezone et vérifier effets

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

