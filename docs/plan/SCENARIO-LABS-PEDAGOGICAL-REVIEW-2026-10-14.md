# Revue pédagogique des consignes LPIC-101 — 14 octobre 2026

## Périmètre et méthode

Relecture statique des **70 labs** `labs/lpic-1-v5/*/*/lab.json` du programme LPIC-101 (y compris le scénario 103.1 conservé pour compatibilité). Pour chaque scénario : titre, mission, critères, concepts rattachés et type de contrôle. Pour les cas problématiques, confrontation avec `setup.sh`, `reference-solution.sh`, les indices et les commandes réellement utilisées par les contrôles.

Questions de revue :
1. **Mission réelle** : un administrateur comprend la panne et le résultat utile attendu, sans exécuter des fichiers témoins arbitraires.
2. **Notion LPIC** : l'activité traite bien au moins un concept déclaré et ne court-circuite pas l'apprentissage.
3. **Consigne complète** : ressources d'entrée, chemins et formats obligatoires sont décrits dans le brief ou les critères.
4. **Vérification loyale** : ne pas présenter de résultat fixe si le setup randomise les données ; décrire les preuves réellement vérifiées.
5. **Lisibilité** : contexte bref, résultats dans les critères, indices progressifs ; ne pas exposer directement la solution dans la mission.

## Bilan

- **70 scénarios relus**, répartis en 101 (19), 102 (21), 103 (17) et 104 (13).
- **18 consignes corrigées** dans cette revue, sur les objectifs 101.2, 101.3, 102.2, 102.6, 103.1, 103.2, 103.3, 103.4, 103.7, 104.1 et 104.7.
- Deux écarts pédagogiques pouvant bloquer l'apprenant : `103.2/incident-report-rebuild` promettait les hôtes fixes `web_02` malgré un nom aléatoire ; `103.4/bulk-argument-dispatch` promettait `beta.txt:11` malgré une taille aléatoire. Les critères sont désormais dynamiques et cohérents avec les commandes de vérification.
- Deux titres en anglais traduits : entrée GRUB alternative et provisionnement NoCloud.
- Une tâche artificielle supprimée : création de `handoff-ready` dans `103.1/transfer-shell-handoff`, y compris son contrôle et les références dans le setup, la solution et les indices.
- Les labs restants ont une mission et un lien identifiable avec leurs concepts. Cette revue de texte **ne vaut pas validation d'exécution des 70 labs**.

## Synthèse par objectif

| Objectif | Labs | Consignes corrigées | Observations |
| --- | ---: | ---: | --- |
| 101.1 | 6 | 0 | Scénarios matériels orientés incident |
| 101.2 | 5 | 1 | Redémarrage réel explicité pour journald |
| 101.3 | 8 | 1 | Redémarrage propre et outil parent clarifiés |
| 102.1 | 3 | 0 | Disques, swap, GPT et LVM reliés à un usage |
| 102.2 | 5 | 2 | Titre GRUB traduit ; redémarrage et diagnostic explicités |
| 102.3 | 2 | 0 | Résolution des bibliothèques dynamique et durable |
| 102.4 | 3 | 0 | Maintenance Debian hors ligne |
| 102.5 | 4 | 0 | RPM, DNF et Zypper reliés aux pannes simulées |
| 102.6 | 4 | 1 | NoCloud traduit, source et état cible clarifiés |
| 103.1 | 6 | 4 | Rôle des shell startup files, rapports et passage aux enfants clarifié ; témoin superflu retiré |
| 103.2 | 2 | 2 | Champs dynamiques et structure du manifeste corrigés |
| 103.3 | 2 | 2 | Entrées/sorties définies ; marqueur READY replacé dans le flux de livraison |
| 103.4 | 2 | 2 | Adaptation stdin → arguments et flux redirigés rendus explicites |
| 103.5 | 2 | 0 | Manipulations des processus réellement observées |
| 103.6 | 1 | 0 | Priorités des processus et mesure réelle |
| 103.7 | 1 | 1 | Fichier d'entrée et numéros aléatoires clarifiés |
| 103.8 | 1 | 0 | Exercice vi interactif lié à l'édition réelle |
| 104.1 | 4 | 1 | Tolérances de tailles des partitions annoncées |
| 104.2 | 2 | 0 | Maintenance de systèmes de fichiers et préservation des données |
| 104.3 | 3 | 0 | Montage, démontage et persistance liés aux incidents |
| 104.5 | 2 | 0 | Deux stratégies de partage distinctes |
| 104.6 | 1 | 0 | Différences pratiques entre liens et copies |
| 104.7 | 1 | 1 | Recherche `find` et format du rapport explicités |

## Recommandations de vérification restantes

- Exécuter les solutions de référence dans leurs environnements Podman, puis les tests KVM pour les scénarios concernés.
- Les contrôles qui ne vérifient **que le résultat final**, par exemple la production de fichiers texte, ne prouvent pas à eux seuls l'utilisation d'un outil précis comme `xargs`, `tee` ou `find`. Éviter de présenter cette preuve comme établie sans trace d'exécution fiable.
- Vérifier en TUI que les titres, missions et critères restent entièrement accessibles sans troncature.
- Préserver la distinction entre une mission contextualisée, les **critères observables** et les **indices** : ne pas faire du brief une suite de commandes.

## Garde-fous éditoriaux pour les prochains labs

Lors de l'ajout d'un scénario, vérifier ensemble `lab.json`, `setup.sh`, `reference-solution.sh` et `hints/*`. Tout attendu calculé sur des données randomisées doit être formulé dynamiquement. Un fichier de sortie imposé doit servir la mission (rapport, log, sauvegarde ou signal consommable), pas seulement faciliter le vérificateur. Les termes LPIC en anglais sont acceptés pour les commandes et objets systèmes, mais les titres et consignes restent rédigés en français.
