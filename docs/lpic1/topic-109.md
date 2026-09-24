# Topic 109 — Fondamentaux réseau

Exam: **102-500**

Comprendre TCP/IP, configurer et diagnostiquer réseau et résolution de noms.

## Objectives

### 109.1 — Fondamentaux TCP/IP

Weight: **4**  
Recommended practice backend: **theory+isolated-network**

**Concepts an agent must teach**
- IPv4 et IPv6
- CIDR/masques/sous-réseaux
- adresses privées/publiques
- TCP vs UDP vs ICMP
- ports et services courants
- boucle locale et routes conceptuelles
- lecture /etc/services

**Examinable terms/files/utilities to cover**

`/etc/services`, `IPv4`, `IPv6`, `subnet`, `CIDR`, `TCP`, `UDP`, `ICMP`, `20`, `21`, `22`, `23`, `25`, `53`, `80`, `110`, `123`, `139`, `143`, `161`, `162`, `389`, `443`, `465`, `514`, `636`, `993`, `995`

**Competence evidence**
- calculer sous-réseau simple
- associer protocoles/ports
- expliquer choix TCP/UDP/ICMP dans diagnostic

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

### 109.2 — Configuration réseau persistante

Weight: **4**  
Recommended practice backend: **libvirt-vm**

**Concepts an agent must teach**
- hostname
- hosts/NSS/resolv.conf
- configuration IP/gateway/DNS
- NetworkManager/nmcli
- interfaces Ethernet/Wi-Fi
- connaissance systemd-networkd
- ifup/ifdown historique
- persistance et ordre résolution

**Examinable terms/files/utilities to cover**

`/etc/hostname`, `/etc/hosts`, `/etc/nsswitch.conf`, `/etc/resolv.conf`, `nmcli`, `hostnamectl`, `ifup`, `ifdown`, `NetworkManager`, `systemd-networkd`

**Competence evidence**
- configurer interface dans réseau isolé
- corriger DNS/gateway/hostname persistant
- expliquer source de vérité selon stack

**Modern/legacy note:** ifup/ifdown sont legacy sur beaucoup de systèmes mais encore au syllabus v5.

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

### 109.3 — Diagnostic réseau

Weight: **4**  
Recommended practice backend: **libvirt-vm**

**Concepts an agent must teach**
- interfaces/adresses/routes via iproute2
- default gateway
- sockets via ss
- connectivité ICMP
- chemin réseau
- tests TCP/UDP avec netcat
- legacy net-tools
- distinction panne L2/L3/DNS/service

**Examinable terms/files/utilities to cover**

`ip`, `hostname`, `ss`, `ping`, `ping6`, `traceroute`, `traceroute6`, `tracepath`, `tracepath6`, `netcat`, `ifconfig`, `netstat`, `route`

**Competence evidence**
- diagnostiquer incident multi-cause
- prouver où le trafic casse
- utiliser outils modernes puis reconnaître anciens équivalents

**Modern/legacy note:** ifconfig/netstat/route sont legacy mais examinables.

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

### 109.4 — Résolution de noms DNS

Weight: **2**  
Recommended practice backend: **libvirt-vm**

**Concepts an agent must teach**
- ordre de résolution NSS
- hosts vs DNS
- resolver configuration
- requêtes A/AAAA/MX/NS courantes
- diagnostic DNS
- systemd-resolved awareness
- différence résolution locale et serveur DNS

**Examinable terms/files/utilities to cover**

`/etc/hosts`, `/etc/resolv.conf`, `/etc/nsswitch.conf`, `host`, `dig`, `getent`, `systemd-resolved`

**Competence evidence**
- isoler panne DNS d’une panne IP
- interroger différents types
- corriger ordre/source résolution dans réseau lab

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

