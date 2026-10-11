# Revue des scénarios LPIC-101 — 12 octobre 2026

## Périmètre et statut

Branche `scenario-migration-1031`. Relecture statique des définitions des **70 labs** de `labs/lpic-1-v5/`, de leurs références de correction, de leurs setups et d'un échantillon d'indices.
La matrice `scenario-coverage.json` référence **69 scénarios** : **20 accepted** et **49 implemented**. Le lab historique `103.1/child-environment-handoff` reste disponible par compatibilité, mais n'est pas dans la matrice.
Sur les **162 concepts** de l'examen 101, **80** sont couverts par un scénario marqué `accepted` et **82** uniquement par des scénarios `implemented`. Il n'existe plus de concept sans scénario déclaré.

Attention : **couverture ne signifie pas acceptation**. Les 49 scénarios `implemented` nécessitent encore une vraie recette ; les statuts déclarés `accepted` ne remplacent pas une vérification indépendante.

## Défauts revus et corrections

- `101.3/indexer-service-recovery` : suppression du faux service `sleep infinity`. L'indexeur produit un résultat réel et le correcteur injecte un événement inédit pour vérifier que l'index évolue.
- `103.5/stuck-worker`, `103.5/transfer-operator-session` : les solutions de référence utilisent désormais un pseudo-terminal interactif, `jobs` et `bg`, plutôt que de contourner cette compétence par un simple `kill -CONT`.
- `102.3/preload-library-recovery` : l'absence d'une bibliothèque LD_PRELOAD ne provoque pas nécessairement un échec d'exécution avec glibc ; le lanceur et le correcteur vérifient désormais sa résolution effective par le chargeur dynamique.
- `104.6/release-link-repair` : test de hard link inter-filesystems sur `/dev/shm` (distinct de `/workspace`) et suppression du rapport artificiel `limitations.txt`.
- `103.8/minimal-editor-repair` : consigne réécrite comme incident et distinction explicite entre état obtenu et interaction clavier.
- `103.3/backup-bundle-recovery` : contrôle du contenu extrait des archives tar/cpio, en plus du seul nom des entrées.
- `103.3/release-tree-recovery` : contrôle des horodatages, permissions et données recopiées.
- `104.1/partition-filesystems` : contrôle des tailles réelles des partitions, auparavant absent.
- `scripts/validate_labs.py` : ajout de `bash -n` pour tous les scripts de labs. Le validateur statique standard reste applicable.

**Correction d'audit :** la suspicion initiale sur `102.5/rpm-file-ownership-audit` était erronée : le JSON décodé contient une vraie tabulation et non la chaîne littérale `\\t`. Aucun changement nécessaire.

## Recette restante (non automatisée ici)

Chaque scénario `implemented` doit satisfaire les portes du projet : setup fonctionnel, état initial en échec, solution de référence qui passe tous les contrôles, une solution alternative plausible, reset rouge reproductible, isolation et timeouts. Les cas de boot, kernel, pilotes, systemd et systèmes de fichiers en VM requièrent la **vraie exécution KVM** ; Podman ne suffit pas.

`102.2/grub-serial-one-time-diagnostics` est un cas d'acceptation **manuelle via console GRUB** : sa référence se termine volontairement avec le code 64. Elle ne doit pas être comptée comme une solution de référence réussie automatiquement.

Les contrôles state-based de `103.8` prouvent une configuration correcte, **pas les frappes exactes dans vi**. Les tests interactifs sont un contrôle complémentaire, et ne doivent pas conduire à imposer un historique de commandes comme substitut.

Aucune fusion `main` ni promotion automatique des 49 scénarios `implemented` sans recette réelle.
