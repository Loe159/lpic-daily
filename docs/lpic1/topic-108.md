# Topic 108 — Services essentiels

Exam: **102-500**

Administrer temps, logs, messagerie locale et impression.

## Objectives

### 108.1 — Heure système et synchronisation

Weight: **3**  
Recommended practice backend: **libvirt-vm**

**Concepts an agent must teach**
- horloge système vs RTC
- UTC
- timezone
- NTP
- chrony
- sources/pools
- inspection état synchronisation
- configuration de base

**Examinable terms/files/utilities to cover**

`/usr/share/zoneinfo`, `/etc/timezone`, `/etc/localtime`, `/etc/ntp.conf`, `/etc/chrony.conf`, `date`, `hwclock`, `timedatectl`, `ntpd`, `ntpdate`, `chronyc`, `pool.ntp.org`

**Competence evidence**
- diagnostiquer dérive/non-sync
- configurer source locale simulée
- expliquer RTC/system clock/timezone

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

### 108.2 — Journalisation système

Weight: **4**  
Recommended practice backend: **libvirt-vm**

**Concepts an agent must teach**
- syslog facilities/priorities/actions
- rsyslog configuration
- journald requêtes/filtres
- persistance/taille/vacuum journal
- interaction journald↔rsyslog
- rotation logs
- logger/systemd-cat
- connaissance syslog-ng

**Examinable terms/files/utilities to cover**

`/etc/rsyslog.conf`, `/var/log`, `logger`, `logrotate`, `/etc/logrotate.conf`, `/etc/logrotate.d`, `journalctl`, `systemd-cat`, `/etc/systemd/journald.conf`, `/var/log/journal`, `syslog-ng`

**Competence evidence**
- retrouver événement via filtres
- router message vers fichier
- mettre rotation/persistance et vérifier après reboot

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

### 108.3 — Bases MTA et messagerie locale

Weight: **3**  
Recommended practice backend: **podman+libvirt-vm**

**Concepts an agent must teach**
- rôle MTA
- aliases
- forward utilisateur
- file d’attente
- interface sendmail compatible
- connaissance Postfix/Sendmail/Exim
- mail local de scripts

**Examinable terms/files/utilities to cover**

`~/.forward`, `sendmail`, `newaliases`, `mail`, `mailq`, `postfix`, `exim`

**Competence evidence**
- mettre alias/forward local
- inspecter queue
- faire envoyer notification locale par script sans déployer serveur Internet

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

### 108.4 — Impression avec CUPS

Weight: **2**  
Recommended practice backend: **libvirt-vm**

**Concepts an agent must teach**
- architecture client/serveur CUPS
- queues/imprimantes locales et distantes
- jobs impression
- configuration de base
- diagnostic queue
- compatibilité commandes BSD/System V de base

**Examinable terms/files/utilities to cover**

`/etc/cups`, `lpr`, `lprm`, `lpq`, `CUPS`

**Competence evidence**
- gérer queue simulée
- annuler/inspecter job
- diagnostiquer imprimante indisponible

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

