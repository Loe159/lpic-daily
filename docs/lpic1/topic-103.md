# Topic 103 — Commandes GNU et Unix

Exam: **101-500**

Construire une aisance opérationnelle avec Bash, texte, fichiers, flux, processus, regex et vi.

## Objectives

### 103.1 — Travail en ligne de commande

Weight: **4**  
Recommended practice backend: **podman**

**Concepts an agent must teach**
- syntaxe shell et séquences de commandes
- variables shell/environnement
- PATH et résolution de commande
- quoting/escaping
- historique
- documentation et identification d’une commande
- informations système de base

**Examinable terms/files/utilities to cover**

`bash`, `echo`, `env`, `export`, `pwd`, `set`, `unset`, `type`, `which`, `man`, `uname`, `history`, `.bash_history`, `quoting`

**Competence evidence**
- résoudre tâches sans GUI
- expliquer expansion/quoting
- corriger PATH/environnement

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

### 103.2 — Filtres et transformation de texte

Weight: **2**  
Recommended practice backend: **podman**

**Concepts an agent must teach**
- flux texte ligne/octet
- sélection de colonnes/lignes
- tri/dédoublonnage
- assemblage et découpage
- transformations simples
- checksums
- lecture/compression en flux

**Examinable terms/files/utilities to cover**

`bzcat`, `cat`, `cut`, `head`, `less`, `md5sum`, `nl`, `od`, `paste`, `sed`, `sha256sum`, `sha512sum`, `sort`, `split`, `tail`, `tr`, `uniq`, `wc`, `xzcat`, `zcat`

**Competence evidence**
- composer un pipeline de transformation
- choisir filtre adapté
- vérifier intégrité avec hash

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

### 103.3 — Gestion de fichiers et archives

Weight: **4**  
Recommended practice backend: **podman**

**Concepts an agent must teach**
- création/copie/déplacement/suppression
- récursivité et globbing
- recherche par métadonnées
- actions find
- archives tar/cpio
- copie bas niveau dd
- compression gzip/bzip2/xz
- détection de type

**Examinable terms/files/utilities to cover**

`cp`, `find`, `mkdir`, `mv`, `ls`, `rm`, `rmdir`, `touch`, `tar`, `cpio`, `dd`, `file`, `gzip`, `gunzip`, `bzip2`, `bunzip2`, `xz`, `unxz`, `globbing`

**Competence evidence**
- organiser/arbitrer fichiers selon contraintes
- construire/restaurer archive
- rechercher puis agir en sécurité

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

### 103.4 — Streams, pipes et redirections

Weight: **4**  
Recommended practice backend: **podman**

**Concepts an agent must teach**
- stdin/stdout/stderr et descripteurs
- redirections écrasement/append
- redirection stderr
- pipelines
- dupliquer sortie avec tee
- transformer stdin en arguments via xargs
- ordre des redirections

**Examinable terms/files/utilities to cover**

`stdin`, `stdout`, `stderr`, `<`, `>`, `>>`, `2>`, `2>&1`, `|`, `tee`, `xargs`

**Competence evidence**
- construire pipelines robustes
- séparer stdout/stderr
- expliquer différence pipe vs substitution/arguments

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

### 103.5 — Processus et jobs

Weight: **4**  
Recommended practice backend: **podman**

**Concepts an agent must teach**
- processus avant/arrière-plan
- jobs du shell
- signaux
- inspection arbres/ressources
- processus survivant à la session
- sélection par nom/PID
- multiplexeurs de terminal

**Examinable terms/files/utilities to cover**

`&`, `bg`, `fg`, `jobs`, `kill`, `nohup`, `ps`, `top`, `free`, `uptime`, `pgrep`, `pkill`, `killall`, `watch`, `screen`, `tmux`

**Competence evidence**
- retrouver et arrêter proprement un processus
- reprendre job suspendu
- faire survivre travail à une déconnexion

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

### 103.6 — Priorités de processus

Weight: **2**  
Recommended practice backend: **podman**

**Concepts an agent must teach**
- nice value et ordonnancement utilisateur
- priorité par défaut
- démarrer avec une priorité
- modifier priorité existante
- observer priorité

**Examinable terms/files/utilities to cover**

`nice`, `renice`, `ps`, `top`

**Competence evidence**
- identifier processus perturbateur
- ajuster priorité sans tuer
- expliquer limites utilisateur/root

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

### 103.7 — Expressions régulières

Weight: **3**  
Recommended practice backend: **podman**

**Concepts an agent must teach**
- regex basiques/étendues
- classes de caractères
- quantificateurs et ancres
- groupes/alternatives
- recherche de lignes
- substitution/transformation avec sed
- différence glob vs regex

**Examinable terms/files/utilities to cover**

`grep`, `egrep`, `fgrep`, `sed`, `regex(7)`, `BRE`, `ERE`

**Competence evidence**
- écrire regex à partir d’un besoin
- diagnostiquer faux positifs
- transformer fichier avec substitution ciblée

**Modern/legacy note:** egrep/fgrep peuvent être présentés comme formes historiques/aliases de grep.

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

### 103.8 — Édition avec vi

Weight: **3**  
Recommended practice backend: **podman-pty**

**Concepts an agent must teach**
- modes normal/insertion/commande
- navigation
- insertion/remplacement
- suppression/copie/collage
- recherche
- sauvegarde/quitter
- éditeur par défaut
- connaissance nano/emacs/vim

**Examinable terms/files/utilities to cover**

`vi`, `vim`, `nano`, `emacs`, `/`, `?`, `h`, `j`, `k`, `l`, `i`, `o`, `a`, `d`, `p`, `y`, `dd`, `yy`, `ZZ`, `:w!`, `:q!`, `EDITOR`

**Competence evidence**
- modifier fichier selon consignes uniquement au clavier
- récupérer d’une erreur de mode
- configurer éditeur par défaut

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

