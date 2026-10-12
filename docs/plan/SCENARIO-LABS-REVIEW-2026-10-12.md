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

## Durcissement suite à la review du 12 octobre

- **101.2 UEFI** : le contrôle final utilise le BootOrder **complet** issu du firmware et le compare à une référence capturée par le runner libvirt côté hôte immédiatement après le setup. Changer `target-order` ou `original-order` dans le guest ne permet plus de changer cette référence. Un reset efface la référence et le nouveau setup en capture une nouvelle.
- **101.3 ACPI** : un contrôle `acpi-button-sent` côté hôte exige que libvirt ait réellement accepté l'injection du bouton virtuel. Il complète les checks `acpid`/journald côté guest ; le marqueur `logger` seul ne peut plus valider le lab. L'attestation est privée à la VM et effacée par reset.
- **103.5 job control** : le host terminal observe Ctrl-Z et la saisie de `jobs`/`bg`, et le shell instrumenté n'enregistre ces commandes que si une tâche stoppée/relancée est visible. Les fichiers d'événements du guest seuls ne suffisent plus.
- **103.8 vi** : passage par un terminal interactif désormais obligatoire. Le host observe l'ouverture de `vi /workspace/app.conf` et la sauvegarde `:wq` ou `:x` en complément de la vérification du fichier. La validation ne prétend toujours pas prouver séparément toutes les frappes de navigation.
- **103.7 regex** : données d'entrée variables à chaque setup et comparaison contre la transformation de l'input plutôt qu'une sortie constante.

Ces vérifications réduisent les contournements trivials, mais ne constituent pas une preuve cryptographique des actions réalisées par l'apprenant root dans sa sandbox. La CI statique/unitaire ne remplace pas la recette KVM.

### Points encore bloquants pour l'acceptation LPIC-101

1. Les **49 scénarios `implemented`** restent hors des statistiques `accepted`. Aucun changement automatique de statut n'est prévu.
2. Les 45 scénarios libvirt nécessitent une recette réelle sur image Fedora reconstruite, y compris une solution alternative, le reset rouge et la vérification d'isolation.
3. Le scénario interactif GRUB nécessite une recette humaine sur console série.
4. L'exercice PCI AHCI de 101.1 ne démontre **pas** un réglage d'activation/désactivation dans BIOS/UEFI. Ne pas déclarer cette compétence firmware acceptée sans scénario probant.
5. L'acceptation de `103.8` repose maintenant sur une preuve d'interaction d'ouverture/sauvegarde, mais ses sous-compétences de navigation/recherche/coller restent à vérifier pédagogiquement en session humaine.
