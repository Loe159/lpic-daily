# Phase 1 — vertical slice sélectionné

Status: **Accepted**

Machine-readable selection: `curriculum/lpic-1-v5/phase1-slice.json`.

## Objectifs

### 103.1 — Travail en ligne de commande

Rôle: foundation.

Pourquoi:
- vrai point d'entrée du graphe;
- teste lesson progressive, recall, variables, PATH, quoting et documentation;
- permet de vérifier que le scheduler n'introduit pas artificiellement un chapitre avancé avant les bases.

### 103.5 — Processus et jobs

Rôle: premier domaine interactif/stateful.

Pourquoi:
- dépend uniquement de 103.1;
- permet de tester PTY, jobs, signaux, process inspection, background/foreground;
- le checker doit observer l'état des processus plutôt que la commande saisie.

### 104.5 — Permissions et ownership

Rôle: premier domaine filesystem/identity.

Pourquoi:
- dépend uniquement de 103.1 dans le graphe hard;
- permet de tester modes octaux/symboliques, owner/group, umask, SGID et sticky bit;
- plusieurs chemins de commandes peuvent produire le même état final.

## Pourquoi pas 103.4 dès le premier slice?

Pipes/redirections restent prioritaires dans le cursus complet, mais les trois objectifs sélectionnés couvrent davantage de surfaces techniques avec seulement trois nodes: shell de base, processus/PTY et filesystem/permissions. `103.4` sera l'un des premiers ajouts après validation de la boucle.

## Concept coverage

Le slice ne crée pas une version simplifiée artificielle de ces objectifs: les concepts listés dans `concepts.json` pour 103.1, 103.5 et 104.5 doivent tous avoir au minimum une stratégie de couverture avant la sortie de Phase 1.

Cela représente actuellement:
- 103.1: 12 concepts;
- 103.5: 7 concepts;
- 104.5: 8 concepts.

Total: **27 concepts**.

Le premier vertical slice peut approfondir certains concepts plus que d'autres, mais aucun des 27 ne doit être invisible dans la matrice de couverture.

## Trois scenarios de référence

### shell-environment-repair — 103.1

Environnement volontairement mal configuré. Le learner doit diagnostiquer pourquoi une commande résout le mauvais binaire / pourquoi une variable ou un quoting produit un résultat inattendu, puis rétablir l'état demandé.

Tests du moteur:
- lesson + recall;
- checker déterministe;
- plusieurs solutions valides;
- debrief.

### stuck-worker — 103.5

Plusieurs jobs/processus existent. Un worker précis doit être identifié, repris ou mis en arrière-plan selon le scénario puis terminé proprement, sans toucher aux autres.

Tests du moteur:
- PTY;
- process state;
- signaux;
- hints progressifs;
- vérification indépendante de la séquence exacte de commandes.

### shared-dropbox — 104.5

Créer/configurer un espace partagé respectant une politique: owner/group, droits dossier/fichiers, SGID/sticky selon variante et comportement attendu pour nouveaux fichiers.

Tests du moteur:
- plusieurs utilisateurs dans la sandbox;
- permission state;
- umask vérifié par création d'un fichier depuis un nouveau shell configuré;
- audit d'un bit SUID inutile;
- state checker owner/mode;
- reset complet.

### Contextes de transfer

Chaque objectif du slice dispose aussi d'un second lab couvrant les mêmes concept IDs dans un état initial différent:
- transfer-shell-handoff pour 103.1;
- transfer-operator-session pour 103.5;
- transfer-team-share-audit pour 104.5.

La matrice de couverture impose au moins deux lab IDs distincts pour chacun des 27 concepts. Après une preuve independent assez ancienne, le planificateur recommande prioritairement le contexte encore inutilisé afin que transfer soit atteignable dans le produit réel.

## Storyboard d'une séance

```text
desktop notification
        |
        v
lpic today
        |
        +--> pourquoi cette séance ?
        |
        +--> 2–5 recalls dus
        |
        +--> micro-lesson d'un concept prêt
        |
        +--> rep rapide
        |
        +--> lab si pertinent / temps disponible
        |       |
        |       +--> hint 1 -> hint 2 -> hint 3 -> solution
        |
        +--> debrief
        |
        +--> append mastery evidence
        +--> append XP/achievement events séparés
        |
        +--> recalcul de la prochaine review
```

## Ordre d'implémentation

1. Parser/valider concepts, graph et content schemas.
2. SQLite migrations: content state, evidence log, gamification events.
3. Projection mastery minimale et explicable.
4. Scheduler utilisant hard prerequisites + due review.
5. TUI: dashboard/session/lesson/question.
6. Podman runner fail-closed.
7. Checker primitives.
8. Scénario `shared-dropbox`.
9. Scénario `stuck-worker`.
10. Scénario `shell-environment-repair`.
11. Daily notification adapter Fedora.
12. E2E + sécurité + reprise après interruption.

## Sortie

La Phase 1 n'est terminée que lorsque les critères de `PHASE-1-ACCEPTANCE.md` passent et que les 27 concepts ont une couverture traçable. Aucun travail libvirt n'est requis pour cette sortie.
