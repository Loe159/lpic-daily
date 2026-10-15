# Audit de formulation des labs scénarisés — 15 octobre 2026

## Portée et méthode

- 70 manifests `labs/lpic-1-v5/*/*/lab.json` présents sur `main` ont été relus (mission, critères, et contrôles associés) ; 69 sont référencés comme scénarios `accepted` dans la matrice.
- Critère de lecture : l'apprenant doit comprendre **quoi** produire ou réparer (sens des données, chemin de livraison nécessaire, format, ordre, quantité, permissions et restrictions) mais doit chercher lui-même **comment** le faire (commandes, options, fichiers de correction, séquences).
- Les règles durables sont décrites dans `AGENTS.md`, `labs/AGENTS.md` et `.agents/skills/author-lab/SKILL.md`.
- Une validation statique des manifests réécrits a été effectuée au moment de leur publication. Les exécutables, conteneurs Podman, VM libvirt et scénarios de correction ne sont pas exécutés dans cet audit.

## Corrections apportées

Les vingt manifests suivants ont été ajustés :

| Lab | Correction |
| --- | --- |
| `101.1/onboard-sata-controller-reenable` | Point de montage exigé rendu explicite |
| `101.1/pci-archive-controller-recovery` | Destination et caractère durable du service explicites |
| `101.1/ramdisk-module-recovery` | Taille minimale, type de système de fichiers et montage attendus |
| `101.1/usb-backup-authorization-recovery` | Destination du volume monté |
| `101.2/dual-firmware-rescue-media` | Destination du support UEFI |
| `102.1/lvm-capacity-recovery` | Noms de groupe/volume exigés par les checks |
| `102.2/bios-mbr-rescue-bootcode` | Destination du support de secours |
| `102.5/rpm-file-ownership-audit` | Fin de la contrainte cachée sur la première ligne du rapport |
| `102.6/nocloud-provisioning-recovery` | Politique d'accès explicite plutôt que « bonnes permissions » |
| `102.6/virtual-disk-network-recovery` | Destination du volume virtuel |
| `103.1/external-command-recovery` | Deux champs du compte rendu mieux définis |
| `103.1/history-handoff-recovery` | Plusieurs syntaxes d'affichage compatibles avec le contrôle de l'historique |
| `103.1/transfer-shell-handoff` | Différence entre architecture matérielle et catégorie Bash |
| `103.2/compressed-manifest-recovery` | Préfixes et syntaxe des empreintes |
| `103.3/release-tree-recovery` | Définition des propriétés de fichiers à préserver |
| `103.5/stuck-worker` | Colonnes et ordre du relevé des processus |
| `103.5/transfer-operator-session` | Colonnes et ordre du relevé de maintenance |
| `103.6/priority-incident` | Séparateur et structure du rapport de priorités |
| `103.8/minimal-editor-repair` | Format de l'inventaire des éditeurs |
| `104.7/fhs-search-recovery` | Format du rapport défini par objectifs sémantiques plutôt que noms de commandes ; checker et solution de référence adaptés (ancien format accepté) |

Pour les 50 autres manifests, cette revue n'a pas identifié d'ambiguïté rédactionnelle du même type nécessitant une modification immédiate.

## Limites et prochaine vérification

Cette revue porte sur la **clarté des missions et des contrats de réussite**, pas sur une démonstration que tous les checkers acceptent toutes les réparations Linux fonctionnellement équivalentes. Certains vérifient encore des détails d'implémentation ; une revue distincte des solutions alternatives, avec tests red/green sur les vrais backends, reste nécessaire. Aucune réussite CI/Podman/KVM n'est déduite du seul audit rédactionnel.
