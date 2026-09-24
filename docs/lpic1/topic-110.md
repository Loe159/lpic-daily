# Topic 110 — Sécurité

Exam: **102-500**

Auditer, réduire la surface d’attaque et utiliser SSH/GPG correctement.

## Objectives

### 110.1 — Audit et tâches de sécurité

Weight: **3**  
Recommended practice backend: **libvirt-vm**

**Concepts an agent must teach**
- audit SUID/SGID
- politiques mot de passe/expiration
- ports/processus ouverts
- fichiers ouverts
- limites ressources
- sessions/historique login
- sudo de base
- recherche d’exposition

**Examinable terms/files/utilities to cover**

`find`, `passwd`, `fuser`, `lsof`, `nmap`, `chage`, `netstat`, `sudo`, `/etc/sudoers`, `su`, `usermod`, `ulimit`, `who`, `w`, `last`

**Competence evidence**
- auditer VM et produire constats factuels
- réduire privilèges sudo
- retrouver service/processus exposé

**Modern/legacy note:** netstat est legacy; enseigner ss en pratique mais conserver reconnaissance.

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

### 110.2 — Durcissement de base

Weight: **3**  
Recommended practice backend: **libvirt-vm**

**Concepts an agent must teach**
- shadow passwords
- désactivation services inutiles
- login interdit via nologin
- socket activation systemd
- xinetd historique
- TCP wrappers historique
- principe surface d’attaque minimale

**Examinable terms/files/utilities to cover**

`/etc/nologin`, `/etc/passwd`, `/etc/shadow`, `/etc/xinetd.d`, `/etc/xinetd.conf`, `systemd.socket`, `/etc/inittab`, `/etc/init.d`, `/etc/hosts.allow`, `/etc/hosts.deny`

**Competence evidence**
- identifier/désactiver service inutile
- configurer accès compte limité
- reconnaître mécanismes legacy sans les recommander comme nouveau design

**Modern/legacy note:** xinetd/TCP wrappers sont historiques; v5 les contient encore.

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

### 110.3 — Cryptographie, SSH et GPG

Weight: **4**  
Recommended practice backend: **libvirt-vm+isolated-network**

**Concepts an agent must teach**
- SSH client/serveur concepts
- clés utilisateur et host keys
- known_hosts/authorized_keys
- ssh-agent
- tunnels local/remote/dynamic et X11 forwarding
- GPG clés/signatures/chiffrement
- révocation
- vérification identité/intégrité

**Examinable terms/files/utilities to cover**

`ssh`, `ssh-keygen`, `ssh-agent`, `ssh-add`, `~/.ssh/id_rsa`, `~/.ssh/id_dsa`, `~/.ssh/id_ecdsa`, `~/.ssh/id_ed25519`, `/etc/ssh/ssh_host_*`, `~/.ssh/authorized_keys`, `ssh_known_hosts`, `gpg`, `gpg-agent`, `~/.gnupg`

**Competence evidence**
- établir auth par clé dans lab
- diagnostiquer host-key mismatch
- créer/vérifier signature et chiffrement GPG
- construire tunnel et expliquer flux

**Modern/legacy note:** DSA/RSA peuvent apparaître dans le syllabus; enseigner préférences cryptographiques modernes séparément.

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

