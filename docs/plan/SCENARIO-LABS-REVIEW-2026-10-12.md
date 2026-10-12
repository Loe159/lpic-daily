# Revue des scénarios LPIC-101 — 12 octobre 2026

**Mise à jour du 12 octobre (après revue) :** par décision du responsable du projet, les **69 scénarios sont désormais publiés (`accepted`)** afin d'être testés au fil de l'apprentissage. Les 49 anciens `implemented` ont été promus dans la matrice et les labels. La suite de cette note retrace les états historiques au moment de la review initiale et non le statut actuel. Publication ≠ confirmation de réussite des tests. La CI récente a révélé un échec de solution de référence `103.5/stuck-worker` sur les deux distributions Podman et un problème de formatage Go ; correctifs poussés, nouvelle CI à vérifier. La recette KVM reste à compléter sans bloquer la publication.


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

## Revue incrémentale après publication

- **Publication :** les 69 scénarios déclarés dans `scenario-coverage.json` et leurs labels `lab.json` sont `accepted`, afin de permettre la pratique progressive. Il ne reste aucune entrée `implemented`. La couverture de 162/162 concepts ne représente **pas** 162 validations d'exécution.
- **CI constatée sur `2061d00` :** `foundation` et `fedora-core` réussis ; `go-core` a échoué sur l'ordre des imports du checker ; les deux tests de conformance Podman ont échoué sur la solution de référence `103.5/stuck-worker`. L'import a été remis au format Go, la fenêtre d'attente des signaux de la référence a été allongée et les sorties de références sont désormais reportées dans les erreurs de CI. Résultat de la nouvelle CI non encore confirmé.
- **Bug de notation `103.8` :** les branches `else if` pouvaient créditer certains concepts `vi` malgré l'absence d'interaction PTY. L'interaction éditeur est dorénavant vérifiée indépendamment, et les six concepts associés échouent si elle manque. Test négatif ajouté.
- **Réponses constantes :** `103.2/incident-report-rebuild`, `103.2/compressed-manifest-recovery`, `103.4/batch-stream-repair` et `103.4/bulk-argument-dispatch` utilisaient des sorties attendues écrites en dur. Les setups introduisent maintenant une variation d'entrée par session et les checks calculent les sorties depuis les vrais fichiers d'entrée. Les solutions restent opérationnelles sur ces variations.
- **Limite résiduelle :** les checks de résultat évaluent l'effet de l'opération, et non l'historique exact des commandes. La preuve d'interaction `vi` reste une vérification d'ouverture/sauvegarde, pas un enregistrement complet de ses six sous-compétences. Le test interactif utilisateur reste utile pour le retour pédagogique, mais ne conditionne plus le statut `accepted`.

## Validation KVM réelle — 12 octobre 2026

Résultat fourni par l'utilisateur après mise à jour de la branche au commit `c653ad6` :

```text
LPIC_DAILY_RUN_KVM_INTEGRATION=1 go test -tags=integration ./internal/runner/libvirt -run '^TestRealKVMPhase2ReferenceLabs$' -count=1 -v -timeout=30m
--- PASS: TestRealKVMPhase2ReferenceLabs (808.54s)
    --- PASS: lpic1.101.1.onboard-sata-controller-reenable
    --- PASS: lpic1.101.1.storage-bus-archive-recovery
    --- PASS: lpic1.101.1.usb-backup-authorization-recovery
    --- PASS: lpic1.101.2.legacy-init-service-migration
    --- PASS: lpic1.101.2.uefi-boot-order-recovery
    --- PASS: lpic1.101.3.acpi-power-event-audit-recovery
    --- PASS: lpic1.101.3.rescue-target-service-recovery
    --- PASS: lpic1.102.2.grub-kernel-parameter
    --- PASS: lpic1.102.2.grub-legacy-migration-recovery
    --- PASS: lpic1.104.1.partition-filesystems
PASS
```

**Statut réel : 10/10 sous-tests KVM passés ensemble**, incluant état initial non résolu, solution de référence, reset, deuxième résolution. Les 45 scénarios libvirt publiés ne sont pas intégralement couverts par cette suite ; ne pas confondre publication et recette complète. Le test d'isolation `TestRealKVMIsolationScenarioAndCrashReaping` n'a pas été inclus dans cette exécution.

### Suivi CI Podman

La dernière CI étudiée (`38247883851`) conserve un échec `103.5/stuck-worker` sur les deux runners Podman. Un correctif ultérieur ajoute un marqueur de disponibilité après installation des traps du processus, afin de ne pas envoyer `SIGTSTP` trop tôt, ainsi que des diagnostics sur échec ; test CI du nouveau commit non encore confirmé.


### Validation isolation KVM sur hôte réel — 12 octobre 2026

Résultat fourni par l'utilisateur :

```text
LPIC_DAILY_RUN_KVM_INTEGRATION=1 go test -tags=integration ./internal/runner/libvirt -run '^TestRealKVMIsolationScenarioAndCrashReaping$' -count=1 -v -timeout=15m
--- PASS: TestRealKVMIsolationScenarioAndCrashReaping (100.37s)
PASS
ok github.com/Loe159/lpic-daily/internal/runner/libvirt 100.391s
```

Ce test complète les **10/10 scénarios de référence KVM** déjà validés. Le test d'isolation porte sur réseau isolé, protections du système hôte, immutabilité de l'image de base, console série, destruction et réconciliation d'une VM simulée orpheline. Cela n'est pas une preuve d'exécution des 45 scénarios libvirt disponibles.
