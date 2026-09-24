# Topic 106 — Interfaces utilisateurs et bureaux

Exam: **102-500**

Comprendre X11/Wayland, environnements de bureau, accès distant et accessibilité.

## Objectives

### 106.1 — Architecture X11 et bases Wayland

Weight: **2**  
Recommended practice backend: **libvirt-vm**

**Concepts an agent must teach**
- client/serveur X11
- DISPLAY
- Xauthority et accès
- configuration Xorg
- clavier/affichage
- display manager/window manager
- X distant
- concepts de base Wayland

**Examinable terms/files/utilities to cover**

`/etc/X11/xorg.conf`, `/etc/X11/xorg.conf.d`, `~/.xsession-errors`, `xhost`, `xauth`, `DISPLAY`, `X11`, `Wayland`

**Competence evidence**
- diagnostiquer DISPLAY/auth X
- interpréter configuration Xorg simple
- expliquer différence architecturale X11/Wayland

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

### 106.2 — Bureaux graphiques et accès distant

Weight: **1**  
Recommended practice backend: **theory+libvirt-vm**

**Concepts an agent must teach**
- rôles desktop environment/window manager/display manager
- caractéristiques générales KDE/GNOME/Xfce
- protocoles d’accès graphique distant
- différences VNC/RDP/SPICE/XDMCP

**Examinable terms/files/utilities to cover**

`KDE`, `GNOME`, `Xfce`, `X11`, `XDMCP`, `VNC`, `SPICE`, `RDP`

**Competence evidence**
- choisir protocole selon scénario
- identifier composant graphique en cause
- expliquer distinction environnement de bureau/protocole distant

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

### 106.3 — Accessibilité du poste Linux

Weight: **1**  
Recommended practice backend: **theory+desktop-sim**

**Concepts an agent must teach**
- contraste/thèmes
- lecteur d’écran
- braille
- magnification
- clavier virtuel
- sticky/slow/bounce/toggle keys
- contrôle souris au clavier
- gestes
- reconnaissance vocale

**Examinable terms/files/utilities to cover**

`screen reader`, `Braille display`, `screen magnifier`, `on-screen keyboard`, `sticky keys`, `repeat keys`, `slow keys`, `bounce keys`, `toggle keys`, `mouse keys`, `gestures`, `speech recognition`

**Competence evidence**
- associer besoin d’accessibilité à fonctionnalité
- configurer exemple dans environnement de test lorsque possible

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

