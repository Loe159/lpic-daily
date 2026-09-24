# Topic 105 — Shells et scripts shell

Exam: **102-500**

Personnaliser Bash et écrire de petits scripts fiables.

## Objectives

### 105.1 — Personnalisation de l’environnement shell

Weight: **4**  
Recommended practice backend: **podman**

**Concepts an agent must teach**
- login shell vs shell interactif
- ordre/finalité fichiers de profil
- variables environnement
- fonctions/aliases
- PATH
- /etc/skel
- configuration globale vs utilisateur

**Examinable terms/files/utilities to cover**

`.`, `source`, `/etc/bash.bashrc`, `/etc/profile`, `env`, `export`, `set`, `unset`, `~/.bash_profile`, `~/.bash_login`, `~/.profile`, `~/.bashrc`, `~/.bash_logout`, `function`, `alias`, `/etc/skel`

**Competence evidence**
- prédire quel fichier est chargé
- mettre en place environnement durable
- corriger PATH/function/alias sans effets globaux

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

### 105.2 — Scripts shell simples

Weight: **4**  
Recommended practice backend: **podman**

**Concepts an agent must teach**
- shebang et exécution
- variables/arguments
- tests et statuts de sortie
- if
- boucles for/while
- lecture entrée
- substitution de commande
- chaînage &&/||
- exec
- emplacement/permissions de scripts
- actions conditionnelles comme mail

**Examinable terms/files/utilities to cover**

`for`, `while`, `test`, `if`, `read`, `seq`, `exec`, `||`, `&&`, `shebang`, `exit status`

**Competence evidence**
- écrire script idempotent simple
- gérer erreurs/conditions
- modifier script existant et tester comportements limites

**Content/lab implications**
- Explain the causal model before memorization-heavy details.
- Include at least one retrieval prompt that can be answered without multiple-choice cues.
- Where the backend permits it, include state-based hands-on evidence rather than exact-command matching.
- Revisit the concept later in a cross-topic incident so transfer is tested.

