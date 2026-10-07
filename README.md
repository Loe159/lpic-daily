<p align="center">
  <img src="assets/lpic-daily-banner.svg" alt="LPIC Daily — Apprends Linux. Pratique chaque jour." width="100%">
</p>

<p align="center">
  <strong>Un entraînement Linux quotidien, local et pratique pour préparer LPIC-1.</strong>
</p>

LPIC Daily transforme l’apprentissage de Linux en une routine courte et progressive : cours ciblés, questions de rappel, exercices pratiques et labs jetables directement depuis le terminal.

Au lieu de simplement mémoriser des commandes, tu manipules de vrais environnements Linux. Les labs spécialisés vérifient directement l’état obtenu ; les ateliers de couverture générale valident une preuve structurée de pratique et restent volontairement classés comme pratique guidée.

## Ce que tu peux faire

- **Apprendre chaque jour** avec une séance adaptée à ta progression.
- **Réviser au bon moment** grâce au suivi local de ta maîtrise.
- **Pratiquer dans de vrais labs** sans risquer de casser ton système.
- **Recevoir une notification quotidienne** et ouvrir directement ta séance.
- **Suivre ta progression** avec XP, streaks et achievements.
- **Travailler hors ligne** : les cours, questions, résultats et progrès restent locaux.

Le contenu est en français, avec les termes Linux conservés en anglais lorsqu’ils sont utilisés ainsi dans la pratique et dans LPIC.

Le parcours embarqué couvre les **42 objectifs actifs de LPIC-1 v5.0**, soit les examens **101-500 et 102-500**. Chaque concept possède un cours ciblé, plusieurs questions dont du rappel libre, et au moins deux contextes de pratique. Les commandes, fichiers, opérateurs et ports explicitement examinables sont expliqués dans les cours et réutilisés dans les exercices.

## Installation

LPIC Daily fonctionne sous Linux. Fedora est actuellement l’environnement le mieux intégré.

Pour compiler le projet, il te faut **Go 1.27+** :

```bash
git clone https://github.com/Loe159/lpic-daily.git
cd lpic-daily
go build -o lpic ./cmd/lpic
./lpic
```

Au premier lancement, LPIC Daily configure automatiquement les éléments utilisateur nécessaires. Lorsqu’une opération nécessite une installation système, un téléchargement ou `sudo`, elle est affichée et demande confirmation.

Tu peux aussi préparer l’environnement explicitement :

```bash
lpic install
```

Si tu ne veux pas installer la partie KVM/libvirt tout de suite :

```bash
lpic install --no-vm
```

Pour mettre ensuite LPIC Daily à jour directement depuis la branche `main` :

```bash
lpic update
```

La commande compile la dernière version de `main` dans un emplacement temporaire puis remplace le binaire utilisateur dans `~/.local/bin/lpic`. Ta progression et tes données locales sont conservées.

> Les labs VM nécessitent également Python 3, QEMU/KVM et libvirt. Sur Fedora, LPIC Daily peut proposer d’installer les dépendances manquantes.

## Utilisation

Lance simplement :

```bash
lpic
```

Si tu as quitté une séance, `lpic continue` rouvre le dashboard à partir de ta progression locale. Un lab interrompu repart dans un environnement jetable propre ; les preuves déjà enregistrées sont conservées.

Tu arrives sur le dashboard de ta séance du jour.

Quelques commandes utiles :

```bash
lpic today              # afficher la séance du jour
lpic today --quick      # séance plus courte
lpic assess             # évaluation initiale
lpic lab list           # voir les labs disponibles
lpic lab run <lab-id>   # lancer un lab
lpic doctor             # vérifier l'environnement
lpic update             # mettre à jour depuis main
```

## Les labs

Les exercices s’exécutent dans des environnements jetables :

- **Podman rootless** pour les exercices de shell, fichiers, permissions et processus ;
- **QEMU/KVM + libvirt** lorsqu’une vraie machine est nécessaire, par exemple pour le boot, le stockage, les services ou le réseau ;
- **Fedora, Debian et openSUSE** lorsque la compétence dépend d’un écosystème de distribution (DNF/RPM, APT/dpkg, Zypper/RPM).

Les images VM sont préparées à la demande : LPIC Daily indique le téléchargement ou la construction nécessaire et demande confirmation avant de le lancer.

Dans un lab :

```text
:shell    ouvrir ou rouvrir le shell persistant
:status   vérifier l'état sans enregistrer de tentative
:check    vérifier et enregistrer ta tentative
:hint     demander l'indice suivant
:reset    recommencer le lab
:quit     quitter et détruire l'environnement
```

Les commandes du lab ne sont jamais exécutées directement sur ton système hôte en fallback.

## Progression et données

Ta progression est stockée localement dans SQLite. Aucun compte ni service cloud n’est nécessaire pour utiliser le cœur de LPIC Daily.

La maîtrise et la gamification sont séparées : gagner de l’XP ou maintenir un streak ne suffit pas à valider une compétence. Les concepts progressent selon les preuves réellement obtenues dans les cours, questions et exercices.

## À propos de LPIC-1

LPIC Daily suit les objectifs LPIC-1 v5.0 des examens **101-500** et **102-500**.

Le projet cherche à couvrir le programme de certification tout en enseignant des pratiques Linux réellement utiles. Les connaissances historiques encore demandées à l’examen sont distinguées des pratiques modernes lorsque c’est nécessaire.

> LPIC Daily est un projet indépendant et n’est pas affilié à Linux Professional Institute. Le contenu pédagogique du projet est original.

## Contribuer

Les contributions sont bienvenues. Consulte [CONTRIBUTING.md](CONTRIBUTING.md) pour commencer.

## Licence

Le code est distribué sous **Apache-2.0**.  
Le contenu éducatif et la documentation originale sont distribués sous **CC BY 4.0**.

Voir [LICENSE](LICENSE) et [LICENSE-CONTENT.md](LICENSE-CONTENT.md).
