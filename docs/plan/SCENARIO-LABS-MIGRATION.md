# Plan — migration vers des labs scénarisés et state-based

Status: **In progress**

Ce document définit la migration du curriculum pratique LPIC-Daily depuis les micro-labs générés par concept vers des scénarios réalistes de diagnostic, réparation et administration Linux.

### Avancement du pilote

Le premier lot implémente la matrice de couverture, son audit CI et le filtrage du scheduler, puis migre 103.1 vers cinq scénarios acceptés couvrant ses 12 concepts. Les scénarios de référence déjà complets font ensuite passer 103.5 (7/7 concepts) et 104.5 (8/8 concepts) dans le même mécanisme. La migration du reste du topic 103 avance avec 103.2 (7/7 concepts) couvert par deux scénarios — reconstruction d’un rapport d’incident et réparation d’un manifeste d’archives compressées — puis 103.3 (8/8 concepts) couvert par une remise en état d’arborescence de release et un bundle de sauvegarde hors ligne. 103.4 (7/7 concepts) ajoute un incident de flux stdout/stderr et une distribution d’arguments depuis stdin. 103.6 (5/5 concepts) est couvert par un incident de priorité de processus fondé sur l’état réel des valeurs nice. 103.7 (7/7) traite un filtrage de logs avec regex/sed, et 103.8 (8/8) une réparation de configuration avec vi. Le topic 103 est ainsi scenario-complete : 61/61 concepts couverts. Les fallbacks restent techniquement chargeables pour compatibilité historique mais ne sont plus proposés lorsqu'un scénario accepté couvre le concept.

Le scénario de référence est "lpic1.103.1.shell-environment-repair": un environnement est volontairement placé dans un état incorrect, l'apprenant doit investiguer librement, corriger le système, puis LPIC-Daily valide le comportement réel obtenu dans un environnement neuf.

## Point d'avancement — 9 octobre 2026 (branche `scenario-migration-1031`)

Couverture LPIC-101 (162 concepts actifs, chiffres issus de la matrice) :
- **80** concepts disposent d'un scénario `accepted` (topic 103 complet et 104.5 à 104.7).
- **24** concepts supplémentaires figurent dans des scénarios `implemented`, **non encore acceptés** (notamment 101.3, 102.3/102.5/102.6, 104.1 à 104.3).
- **58** concepts n'ont pas encore de scénario `accepted` ou `implemented`.

Nouveaux scénarios de ce lot :
- `104.2.offline-ext-audit-recovery` : contrôle, réparation et réglage réels d'une image ext4 hors ligne (Podman, 3 concepts) ;
- `104.3.archive-mount-recovery` : restauration d'un vrai montage et de la persistance fstab/UUID (KVM, 4 concepts) ;
- `101.3.default-target-recovery` : correction de la cible systemd par défaut (KVM, 1 concept) ;
- `101.3.indexer-service-recovery` : remise en marche d'un vrai service systemd en échec (KVM, 1 concept) ;
- `104.2.inode-backlog-recovery` : remplacement de la pseudo-saturation Podman par un volume ext4 KVM dont les inodes sont effectivement épuisés (2 concepts).
- `101.3.indexer-service-recovery` : correction d'un service systemd défaillant (KVM, 1 concept).

Les instructions des agents sont harmonisées dans `AGENTS.md`, `labs/AGENTS.md`, `curriculum/AGENTS.md`, `.agents/skills/author-lab/SKILL.md` et `docs/CONTRIBUTING_WITH_AGENTS.md`. **Incident réel, symptomatique, mission de récupération et preuve d'état** sont désormais le modèle exigé. Les scénarios ne peuvent pas être considérés `accepted` avant exécution du setup, test négatif, solution de référence, reset, review pédagogique et contrôle du backend. Ce lot est un travail d'**implémentation**, sans revendication de tests Podman/KVM ou de CI verte.

Prochaines zones de migration : complément 104.1–104.3 (avec KVM pour les systèmes réels), 102.1–102.6 (APT/Dpkg, RPM/DNF, boot, libs et virtualisation), puis 101.1–101.3 (découverte matériel, démarrage et services). Ne pas déclarer tout le 101 migré tant que la preuve 162/162 n'est pas établie.

### Lot supplémentaire — 9 octobre 2026 (migration du topic 104 et APT)

Cinq nouveaux incidents `implemented` (à exécuter sur les VM réelles avant acceptation) :
- `104.3.busy-removable-volume` : diagnostiquer et libérer un volume ext4 occupé par un processus, puis le démonter proprement ;
- `104.3.mount-unit-incident` : réparer la source d'une unité systemd `.mount` et retrouver un manifeste existant ;
- `104.1.xfs-archive-label-recovery` : identifier une erreur d'étiquette et récupérer un volume XFS sans reformater ; couvre également les outils de maintenance hors ligne de `104.2` ;
- `104.1.btrfs-pool-report-recovery` : restaurer un sous-volume sur un vrai pool Btrfs à deux disques et réactiver la compression ;
- `102.4.offline-apt-repository-recovery` : restaurer une source APT `file:` et installer un vrai paquet dans une VM Debian, sans connexion Internet.

Couverture matricielle LPIC-101 après ce lot : **80 acceptés, 35 uniquement implémentés, 47 sans scénario**, total **162**. Les objectifs `104.2` et `104.3` sont entièrement **implémentés (7/7)** mais ne sont **pas acceptés** ; `104.1` a 7/8 concepts implémentés. Ce compteur ne valide ni le comportement du runner ni la qualité pédagogique. Il reste à exécuter chaque setup, les checks négatifs, la solution de référence, le reset et l'essai en conditions réelles. En particulier, KVM/Podman et la CI n'ont pas été exécutés pour ces ajouts.

### Lot suivant — 9 octobre 2026 (formats, paquets Debian et chargeur dynamique)

Quatre incidents supplémentaires sont `implemented` :
- `104.1.removable-media-formats` : deux disques de remplacement réels, récupération des montages VFAT/FAT32 et exFAT ; la recette Fedora ajoute `exfatprogs`.
- `102.4.relay-package-dependency-repair` : paquet Debian laissé `unpacked` par une dépendance absente, archive de dépendance locale à installer et agent à configurer, avec inventaire `dpkg` vérifié.
- `102.4.collector-config-recovery` : paquet Debian installé mais configuration corrompue ; phase de reconfiguration exploitable sans toucher aux données ; la recette Debian ajoute `debconf`.
- `102.3.shared-library-cache-recovery` : programme ELF lié à une bibliothèque fournisseur déplaçée, réparation de la recherche système et du cache `ldconfig` ; la recette Fedora ajoute les outils de compilation nécessaires au setup (`gcc`, `glibc-devel`).

État de couverture LPIC-101 après le lot, selon `scenario-coverage.json` : **80/162 acceptés**, **41/162 supplémentaires implémentés sans validation runtime**, **41/162 non encore couverts**. Les objectifs **104.1 (8/8)**, **102.3 (5/5)** et **102.4 (7/7)** sont maintenant entièrement implémentés mais **pas acceptés**. Contrôle statique du mapping concepts/checks et des métadonnées des quatre nouveaux labs : aucune incohérence. **Ni CI ni tests réels Podman/KVM n'ont été exécutés pour ces scénarios**. Les recettes d'image modifiées devront être reconstruites et revérifiées avant l'acceptation.

### Complément du lot — 9 octobre 2026 (extension LVM)

`102.1.lvm-capacity-recovery` est désormais `implemented` : un disque additionnel doit être intégré comme partition/PV, VG et LV pour rétablir le point de montage ext4 de l'application. La recette VM Fedora inclut `lvm2`. La validation doit porter sur les véritables volumes bloc et le montage, et non sur un fichier de preuve.

**Couverture matricielle actuelle LPIC-101 : 80/162 concepts acceptés, 43/162 uniquement implémentés, 39/162 sans scénario.** Ce compteur résulte de la matrice et des fichiers de concepts ; il ne représente pas un taux de réussite aux tests. Les cinq scénarios de ce lot ont passé une revue statique de leurs correspondances concept/check/backend/labels, sans anomalie constatée. Les quatre niveaux d'indices, setups, références et validations réelles nécessitent toujours l'exécution de la suite de tests. **CI et essais de VM non exécutés à cette étape.**

Note sur les images : la recette Fedora doit maintenant être reconstruite avec `exfatprogs`, `gcc`, `glibc-devel` et `lvm2`, et la recette Debian avec `debconf`, avant les tests de lab correspondants.

### Lot suivant — 9 octobre 2026 (partitionnement, démarrage, udev et DNF)

Nouveaux scénarios `implemented` :
- `102.1.home-log-swap-isolation` : répartir les données de `/home` et les journaux applicatifs sur de vrais volumes ext4 distincts, activer du swap dimensionné et conserver les données ; contrôles de montages, fstab et swap actifs ;
- `102.1.uefi-rescue-media-recovery` : réparer la signature GPT d'une partition EFI existante sans la reformater, monter le support et vérifier la préservation d'un véritable exécutable EFI ;
- `101.1.backup-device-udev-recovery` : identifier un périphérique de stockage par ses propriétés udev et restaurer un lien stable dans `/dev` ; vérification de la base udev et du service client ;
- `102.5.offline-dnf-agent-recovery` : réparer une source DNF `file://` hors ligne et installer des RPM réels avec résolution de dépendance ; la recette VM Fedora ajoute `rpm-build` et `createrepo_c`.

Scénario existant désormais référencé dans la matrice :
- `102.2.grub-kernel-parameter` : le marqueur artificiel `lpic_daily_boot` a été remplacé par des paramètres réels `loglevel=7` et `systemd.show_status=yes` ; les checks portent sur la configuration BLS Fedora et les paramètres observés **après redémarrage**, avec un rattachement explicite au concept 101.2 des paramètres noyau au boot. Le scénario requiert la commande de redémarrage du lab en fin de réparation.

**Couverture LPIC-101 de la matrice au 9 octobre 2026** : **80/162 concepts acceptés**, **54/162 uniquement implémentés sans acceptation**, **28/162 concepts sans scénario**. **102.1 est implémenté à 6/6** ; le topic 102.2 ne couvre encore que 2/6, 101.1 2/8 et 102.5 5/7. Une vérification statique des cinq définitions (correspondance entre objectifs, concepts et checks ; cohérence des backends et statuts ; présence de quatre références d'indices) ne signale aucune incohérence. **Aucune validation KVM ou CI n'a été effectuée**. Les recettes VM modifiées nécessitent leur reconstruction avant l'exécution des nouveaux scénarios, et toutes les preuves marquées `implemented` restent à éprouver (état initial rouge, référence verte, reset et review pédagogique).

### Lot suivant — 9 octobre 2026 (journaux de boot et intégrité RPM)

Deux incidents réels supplémentaires sont `implemented`, **sans acceptation runtime** :
- `101.2.boot-journal-persistence` : le serveur perd ses journaux entre deux redémarrages ; correction de `systemd-journald`, transfert des événements encore volatils et preuve après `:reboot` que le journal du démarrage précédent est lisible ;
- `102.5.rpm-integrity-recovery` : un paquet réellement installé est altéré et une des deux archives de restauration est corrompue. Vérification RPM (`rpm -V`, `rpm -K`) et signature détachée GPG (`gpgv`) puis restauration par RPM. La signature détachée est une fixture pédagogique, **pas** une signature RPM native incorporée : vérifier ce périmètre lors de la revue.

**Couverture LPIC-101 d'après la matrice** : **80/162 acceptés, 56/162 uniquement implémentés, 26/162 sans scénario**. Le check du premier lab exige un nouveau boot-id ; sa référence ne peut devenir verte qu'après un reboot réel de la VM. Le second lab exige `rpm-build` et `gnupg2` dans la recette Fedora déjà déclarée. Le setup, le test rouge, la référence, le reset et l'isolation doivent être exercés dans KVM avant d'annoncer l'acceptation. CI non exécutée dans ce lot.

### Lot complémentaire — 9 octobre 2026 (Zypper openSUSE)

`102.5.offline-zypper-repository-recovery` est **implemented**, sans validation runtime. Une VM openSUSE Leap 16.0 dispose d'un dépôt RPM hors ligne local volontairement mal référencé ; l'apprenant doit réparer la source Zypper et installer le collecteur avec sa dépendance réelle, sans réseau. La recette openSUSE ajoute `rpm-build` et `createrepo_c` pour la préparation des paquets d'exercice. La recette VM doit être reconstruite avant l'essai.

**Couverture LPIC-101 : 80/162 acceptés, 57/162 uniquement implémentés, 25/162 non couverts.** L'objectif 102.5 atteint **7/7 concepts implémentés**, mais il n'est pas accepté. Les tests à effectuer sont : build de l'image, setup → checks rouges → solution Zypper → checks verts → reset rouge, contrôle de l'isolation et revue pédagogique. Ni CI ni test KVM n'ont été exécutés pour ce lot.

### Lot complémentaire — 9 octobre 2026 (contrôleur PCI de stockage)

`101.1.pci-archive-controller-recovery` est **implemented**, non testé en KVM. Le setup prépare un vrai disque virtio secondaire avec données conservées, désassocie **uniquement** sa fonction PCI du pilote dans la VM éphémère, puis déclenche un échec observable d'un service systemd. La récupération doit restaurer la visibilité dans `/proc`, `/sys`, `/dev`, le rattachement PCI et l'accès au volume intact. La recette Fedora ajoute `pciutils` pour `lspci` ; reconstruction nécessaire.

**Couverture LPIC-101 : 80 acceptés, 58 uniquement implémentés, 24 sans scénario.** Nouveau concept explicitement démontré : le rôle de /proc, /sys et /dev ; l'identification de ressources matérielles est renforcée. Cet incident PCI ne prouve pas à lui seul l'objectif complet « bus PCI et USB » ni celui de l'activation des périphériques intégrés au firmware : ils restent **non couverts**, de même que la manipulation USB, les modules spécifiques et les familles de stockage. Validation restante : setup rouge, solution verte, reset rouge, absence d'impact sur le disque système et tests libvirt/CI. Rien de cela n'est revendiqué comme exécuté.

### Lot complémentaire — 9 octobre 2026 (module noyau et maintenance)

Deux incidents KVM supplémentaires **implemented**, non acceptés :
- `101.1.ramdisk-module-recovery` : rétablir un vrai module `brd`, reconstruire un système de fichiers RAM éphémère et récupérer un cache depuis une source durable ; la recette Fedora ajoute `kmod` et `kernel-modules-extra`. Ne revendique pas la connaissance de toute l'interface USB.
- `101.3.graceful-maintenance-transition` : arrêter proprement un vrai worker systemd via une cible applicative, prouver le traitement de SIGTERM et préserver QEMU Guest Agent. Ne revendique ni `shutdown` réel ni modes SysV.

**Couverture LPIC-101 : 80 acceptés, 60 uniquement implémentés, 22 sans scénario.** Les deux nouveaux scénarios nécessitent la reconstruction de l'image Fedora et l'exécution setup rouge → référence verte → reset rouge ; aucune CI ni KVM n'est déclarée exécutée.

### Lot complémentaire — 9 octobre 2026 (GRUB BLS et cloud-init)

Deux nouveaux scénarios `implemented`, sans validation runtime :
- `102.2.grub-alternate-entry-recovery` : entrée BLS réellement sélectionnée via GRUB 2, avec un nouveau boot-id et l'option de diagnostic observée dans /proc/cmdline après `:reboot`. N'implique pas la console interactive GRUB.
- `102.6.nocloud-provisioning-recovery` : user-data NoCloud mal formé dans une VM Debian ; réparation, rejeu réel des phases cloud-init et vérification du module write_files. La recette Debian ajoute `cloud-init` : reconstruire l'image.

**Couverture LPIC-101 : 80 acceptés, 62 uniquement implémentés, 20 sans scénario.** La CI, les essais de reboot/VM, les checks négatifs, la solution de référence et les resets restent à exécuter avant toute acceptation. Le lab GRUB doit être relancé via le terminal parent pour la vérification du boot.

### Lot complémentaire — 9 octobre 2026 (récupération des ressources virtuelles)

`102.6.virtual-disk-network-recovery` est **implemented** : un disque ext4 secondaire avec son journal intact et une interface NetworkManager déconnectée doivent être remis en service sur un véritable réseau libvirt **isolé**, sans Internet. La recette Fedora ajoute explicitement `NetworkManager`, et doit être reconstruite avant l'essai.

**Couverture LPIC-101 : 80 acceptés, 63 uniquement implémentés, 19 sans scénario.** Ce lab n'est pas une preuve de pilotes de périphériques, de conteneurs ou de cloud-init : il couvre uniquement les ressources virtuelles de stockage et de réseau. Les tests VM, les états rouge/vert/reset, la sécurité du réseau et la CI restent à effectuer.

## 1. Pourquoi cette migration

Le curriculum contient actuellement 309 concepts actifs:
- Exam 101: 162 concepts;
- Exam 102: 147 concepts.

La couverture exhaustive a été obtenue en générant deux contextes de pratique par concept, soit 618 micro-labs. Cette stratégie garantit qu'aucun concept n'est absent de la progression, mais elle optimise la couverture mécanique plutôt que la qualité de la pratique.

Les labs de référence du vertical slice montrent un meilleur modèle:
- "shell-environment-repair";
- "stuck-worker";
- "shared-dropbox";
- leurs contextes de transfert.

Ces labs mettent l'apprenant face à un système dans un état crédible, permettent plusieurs chemins de résolution et valident l'état final plutôt que la séquence de commandes saisie.

La migration doit donc changer l'unité de conception:

    avant:
    1 concept -> 2 micro-labs

    cible:
    plusieurs concepts cohérents -> 1 scénario réaliste
                               -> checks d'état/comportement
                               -> preuve pratique par concept

La quantité de labs n'est pas un objectif. La couverture, la qualité de la preuve et la variété des situations le sont.

## 2. Objectif produit

À terme, LPIC-Daily doit proposer environ 50 à 70 scénarios principaux pour l'ensemble de LPIC-1, au lieu de 618 micro-labs.

Ordre de grandeur visé:
- Exam 101: environ 27 à 35 scénarios;
- Exam 102: environ 27 à 36 scénarios;
- certains concepts importants peuvent être revus dans plusieurs scénarios;
- certains scénarios peuvent couvrir plusieurs objectifs lorsque la situation réelle le justifie.

Ces nombres sont des budgets, pas des quotas. Un scénario ne doit jamais absorber artificiellement des concepts uniquement pour réduire le nombre total de labs.

## Règle éditoriale obligatoire — consignes orientées incident

Toute consigne de lab doit décrire un contexte opérationnel, un symptôme observable, une mission exprimée en état ou comportement attendu et les contraintes de sécurité ou de conservation. Elle ne doit pas fournir une procédure, une liste ordonnée de manipulations, les commandes à utiliser ou les chemins précis à découvrir. Les critères de validation restent testables, mais n'imposent pas le chemin de résolution. Les indices progressifs portent les détails techniques nécessaires, et la solution de référence reste séparée. Cette règle s'applique à la migration et aux futures contributions ; toute consigne de type checklist de commandes doit être réécrite avant acceptation.

## 3. Principe pédagogique

Un vrai lab LPIC-Daily suit cette boucle:

    état initial volontairement incorrect
                    |
                    v
             symptôme observable
                    |
                    v
             investigation libre
                    |
                    v
              correction réelle
                    |
                    v
                 :check
                    |
                    v
       validation de l'état/comportement
                    |
                    v
      preuves pratiques liées aux concepts

Le brief décrit:
- le contexte;
- le symptôme;
- le résultat attendu;
- les contraintes importantes.

Le brief ne doit normalement pas donner:
- la commande à utiliser;
- le fichier précis à modifier si sa découverte fait partie de la compétence;
- l'ordre exact des actions;
- une procédure déguisée en consigne.

L'apprenant doit pouvoir choisir un chemin différent de la solution de référence et réussir si l'état final est correct.

## 4. Règles de couverture

### 4.1 Couverture minimale

Chaque concept actif doit être:
1. enseigné;
2. évalué par quiz/recall;
3. pratiqué dans au moins un scénario pertinent;
4. relié à au moins un check ou une preuve explicite.

Un concept n'a pas besoin d'avoir son propre lab.

### 4.2 Taille d'un scénario

Cible normale:
- 3 à 8 concepts cohérents;
- possibilité d'aller au-delà lorsque le scénario est naturellement transversal;
- possibilité de rester à 1 ou 2 concepts lorsqu'une compétence est suffisamment importante ou spécifique.

Le regroupement doit suivre la réalité d'administration Linux, pas la structure artificielle du fichier de curriculum.

### 4.3 Profondeur de pratique

Les concepts à forte valeur opératoire ou poids d'examen élevé doivent idéalement être rencontrés dans au moins deux contextes indépendants au cours de la préparation.

Cela ne signifie plus "deux labs par concept". Le second contexte peut être:
- un autre scénario;
- une variante du même scénario;
- un incident cross-topic;
- une révision pratique tardive.

### 4.4 Concepts théoriques

Tous les concepts ne se prêtent pas à "réparer une machine".

Pour les notions de choix, architecture, compatibilité ou connaissance historique:
- construire un scénario de diagnostic ou de décision;
- fournir des observations réelles lorsque possible;
- demander un artefact vérifiable seulement lorsqu'il correspond à une vraie tâche d'administration;
- ne jamais revenir à un fichier artificiel de type CONCEPT/TERMS/EXPLANATION.

Exemples:
- identifier BIOS versus UEFI depuis les artefacts du système;
- choisir un protocole d'accès distant selon des contraintes;
- comparer SysV init, Upstart et systemd dans un scénario de migration;
- diagnostiquer un environnement graphique sans imposer une modification dangereuse.

## 5. Hiérarchie des preuves

Les checks doivent privilégier, dans cet ordre:

1. **Comportement observable**
   - la commande/service/application fonctionne dans le contexte demandé;
   - un autre utilisateur obtient le bon résultat;
   - un nouveau shell ou un nouveau boot observe le bon état.

2. **État réel du système**
   - permissions;
   - ownership;
   - montage;
   - processus;
   - route;
   - configuration persistante;
   - package installé;
   - service actif;
   - contenu ou métadonnées d'un fichier.

3. **Artefact produit**
   - archive valide;
   - fichier de configuration valide;
   - script exécutable;
   - rapport issu d'un pipeline;
   - clé ou signature utilisable.

4. **Décision démontrée**
   - uniquement lorsqu'aucun état machine pertinent n'existe;
   - doit être contrainte et vérifiable;
   - ne doit pas se réduire à "explique ce concept".

5. **Historique de commandes**
   - seulement comme signal secondaire;
   - jamais comme preuve principale si un état final est observable.

Un lab ne doit pas échouer simplement parce que l'apprenant a utilisé une commande différente de la solution de référence.

## 6. Contrat d'un scénario

Chaque scénario doit contenir au minimum:

### Définition
- ID stable;
- titre;
- objectifs;
- concepts couverts;
- durée estimée;
- backend;
- contexte de pratique;
- brief;
- critères de réussite;
- debrief.

### Environnement
- image ou VM versionnée;
- setup déterministe;
- ressources plafonnées;
- réseau désactivé par défaut;
- chemins/ressources modifiables explicitement définis;
- reset complet.

### Validation
- checks déterministes;
- chaque check relié aux concept IDs qu'il prouve;
- au moins un check significatif doit échouer juste après setup;
- tous les checks doivent passer après la solution de référence;
- aucun check ne doit dépendre d'un détail non annoncé ou aléatoire.

### Aide
Quatre niveaux d'indice standardisés:

1. quoi observer;
2. quelle zone du système examiner;
3. quel mécanisme est probablement impliqué;
4. procédure presque complète.

Le quatrième indice peut révéler la solution et doit être marqué comme tel dans l'evidence.

### Référence
- reference-solution.sh ou équivalent;
- utilisée par la CI;
- jamais nécessaire pour exécuter le lab;
- une seule solution de référence ne signifie pas qu'une seule solution utilisateur est acceptée.

## 7. Backends

### Podman

À utiliser lorsque le comportement est correctement représenté dans un conteneur rootless:
- shell;
- fichiers;
- permissions;
- utilisateurs/groupes simulables;
- outils texte;
- archives;
- processus;
- packages lorsque l'image le permet;
- logs ou services simulables de façon fiable.

### Libvirt/QEMU/KVM

À utiliser lorsque le concept dépend réellement d'un système complet:
- bootloader;
- kernel/initramfs;
- partitionnement;
- création/réparation de filesystems bloc;
- démarrage systemd;
- configuration réseau persistante nécessitant reboot;
- scénarios multi-machine lorsque l'isolation réseau du runner est nécessaire.

Le backend ne doit jamais être choisi uniquement pour rendre le lab plus spectaculaire.

## 8. Scheduler et progression

Le scheduler doit évoluer d'un modèle concept-centrique vers un modèle de scénario prêt à jouer.

### Disponibilité

Un scénario devient éligible lorsque:
- ses prérequis d'objectif sont satisfaits;
- les concepts nécessaires à sa compréhension ont été introduits;
- il n'est pas pédagogiquement trop tôt;
- le backend requis est disponible.

Tous les concepts d'un scénario n'ont pas forcément besoin d'être maîtrisés avant le lab: le lab fait partie de leur apprentissage.

### Sélection

Priorités proposées:
1. scénario requis pour débloquer une section;
2. scénario couvrant plusieurs concepts vus mais pas encore pratiqués;
3. scénario de révision dû;
4. scénario de transfert sur un concept faible;
5. cross-topic incident.

### Evidence

Le scheduler et la projection de mastery doivent distinguer:
- ancienne pratique guidée/générée;
- scénario state-based;
- scénario de transfert;
- aide importante ou solution révélée.

Les anciennes evidences restent dans l'historique et ne doivent pas casser les streaks ou la progression existante.

À terme, la readiness LPIC doit pouvoir exiger une quantité minimale de preuve state-based sans effacer les résultats historiques.

## 9. Matrice machine-readable

Ajouter une source de vérité dédiée, proposée sous:

    curriculum/lpic-1-v5/scenario-coverage.json

Elle doit permettre de répondre automatiquement à:
- quel scénario couvre ce concept ?
- quel check le prouve ?
- quel backend est nécessaire ?
- le scénario est-il designé, implémenté ou accepté ?
- le concept dépend-il encore d'un fallback généré ?
- existe-t-il un contexte de transfert ?

Champs minimaux proposés:
- scenario_id;
- objective_ids;
- concept_ids;
- backend;
- status: planned / implemented / accepted;
- evidence_strength: behavior / state / artifact / decision;
- transfer_group éventuel.

Ne pas dupliquer les checks complets dans cette matrice: les fichiers lab restent la source de vérité de l'implémentation.

## 10. Audit automatisé

Ajouter un audit de couverture scénario, par exemple:

    scripts/audit_scenario_coverage.py

Il doit afficher au minimum:
- concepts actifs;
- concepts couverts par un scénario accepté;
- concepts couverts uniquement par fallback généré;
- concepts sans pratique;
- répartition behavior/state/artifact/decision;
- nombre de scénarios;
- nombre moyen et maximal de concepts par scénario;
- concepts n'ayant qu'un seul contexte;
- objectifs entièrement migrés.

La CI doit échouer si:
- un concept perd toute pratique pendant la migration;
- un scénario "accepted" référence un concept inexistant;
- un scénario accepté n'a aucun check lié à un de ses concepts;
- un scénario peut déjà passer intégralement juste après setup;
- la solution de référence ne fait pas passer tous les checks;
- un fallback réapparaît pour un concept marqué migré.

## 11. Tests obligatoires par scénario

Chaque scénario accepté doit avoir:

### Test positif
    setup
    -> reference solution
    -> :check
    -> succès complet

### Test négatif
    setup
    -> :check
    -> échec d'au moins un critère essentiel

### Test de reset
    setup
    -> modifications
    -> reset
    -> état initial reproductible

### Test d'alternative
Lorsque raisonnable, au moins une solution différente de la référence doit être couverte par un test de non-surspécification.

Exemple:
- permissions correctes obtenues en symbolique ou en octal;
- pipeline construit avec un ordre différent mais produisant le même artefact;
- service réparé sans exiger une commande précise.

### Test sécurité/isolation
Pour les scénarios sensibles:
- aucun accès host non autorisé;
- aucune capacité supplémentaire inutile;
- réseau borné;
- chemins modifiables bornés;
- timeout et PID/mémoire limités.

## 12. Standard UX

Le brief doit rester court et lisible dans un terminal.

Cible:
- contexte: 1 à 3 phrases;
- symptômes: liste courte;
- résultat attendu: liste courte;
- aucun dump de théorie;
- aucun détail interne de checker;
- aucune preuve artificielle demandée à l'utilisateur.

Commandes UX communes:
- :check;
- :hint;
- :reset;
- :quit.

Le debrief arrive après réussite et explique:
- pourquoi le problème se produisait;
- quels mécanismes LPIC étaient impliqués;
- quelles autres solutions auraient été valides;
- quel piège d'examen ou d'administration retenir.

## 13. Portfolio cible — Exam 101

Le catalogue ci-dessous est une proposition de design. Le nombre exact peut changer après audit des concepts et prototypage.

### Topic 101 — architecture système: 4 à 6 scénarios

#### 101.1 — Matériel et périphériques
Scénarios candidats:
- périphérique présent mais non exploitable: inspection /proc, /sys, /dev, lspci/lsusb et modules;
- ajout d'un périphérique avec module absent ou mauvais diagnostic udev/sysfs.

#### 101.2 — Chaîne de démarrage Linux
Scénarios candidats:
- noyau présent mais mauvais paramètre de boot;
- initramfs incohérent ou entrée de démarrage à diagnostiquer;
- machine qui atteint un shell de secours au lieu du target attendu.

#### 101.3 — Targets, runlevels, arrêt et redémarrage
Scénarios candidats:
- serveur démarré dans le mauvais target;
- migration d'une procédure runlevel vers systemd;
- arrêt/redémarrage propre avec service bloquant.

### Topic 102 — installation et packages: 6 à 8 scénarios

#### 102.1 — Conception du partitionnement
- machine à installer avec contraintes /, /home, swap, EFI et capacité;
- extension future/LVM à anticiper sans surpartitionner.

#### 102.2 — Bootloader
- configuration GRUB pointant vers le mauvais noyau;
- entrée de boot perdue après changement de disque ou UUID.

#### 102.3 — Bibliothèques partagées
- binaire qui ne démarre plus à cause d'une bibliothèque introuvable;
- cache/chemin de librairies incohérent à diagnostiquer avec ldd/ldconfig.

#### 102.4 — Paquets Debian
- dépendance cassée après installation interrompue;
- package/fichier de configuration à retrouver, vérifier et reconfigurer.

#### 102.5 — RPM, DNF/YUM et Zypper
- package incohérent ou fichier modifié à auditer;
- dépôt ou cache package défectueux;
- comparaison d'opérations RPM bas niveau et gestionnaire haut niveau.

#### 102.6 — Virtualisation et cloud
- guest cloné avec identité/configuration à corriger;
- distinguer informations hyperviseur, VM, conteneur et cloud-init dans un incident.

### Topic 103 — commandes GNU/Linux: 10 à 12 scénarios

#### 103.1 — Ligne de commande
Référence:
- shell-environment-repair.

Compléments:
- environnement transmis incorrectement à un processus enfant;
- commande différente selon utilisateur/contexte, avec alias/fonction/PATH;
- script ou séquence shell cassée par quoting/expansion.

#### 103.2 — Filtres et transformation de texte
- rapport généré depuis des logs contenant colonnes, doublons et valeurs à trier;
- transformation reproductible avec cut, paste, sort, uniq, tr, sed ou équivalents autorisés.

#### 103.3 — Fichiers et archives
- arborescence de release à copier/déplacer sans perdre ce qui compte;
- backup incomplet ou archive incorrecte à reconstruire et vérifier.

#### 103.4 — Streams, pipes et redirections
- tâche batch qui mélange stdout/stderr et produit un rapport invalide;
- pipeline dont une redirection écrase ou envoie les données au mauvais endroit.

#### 103.5 — Processus et jobs
Références:
- stuck-worker;
- transfer-operator-session.

#### 103.6 — Priorités
- processus CPU intensif qui pénalise un service;
- corriger priorité/niceness puis vérifier l'effet sans tuer le mauvais processus.

#### 103.7 — Expressions régulières
- extraire précisément des événements dans un log bruité;
- regex trop large entraînant une action sur de mauvaises lignes.

#### 103.8 — vi
- fichier de configuration à corriger dans un environnement minimal où vi est l'éditeur garanti;
- navigation, recherche, modification et sauvegarde vérifiées par le résultat final, pas par les touches saisies.

### Topic 104 — devices, filesystems et FHS: 7 à 9 scénarios

#### 104.1 — Partitions et filesystems
- disque neuf à préparer suivant une politique donnée;
- table de partitions/filesystem incorrects à identifier puis corriger dans une VM jetable.

#### 104.2 — Intégrité et maintenance
- filesystem plein/inodes épuisés;
- filesystem marqué incohérent à inspecter et réparer hors ligne lorsque nécessaire.

#### 104.3 — Montage
- service cassé après reboot à cause d'un /etc/fstab incorrect;
- mount temporaire fonctionnel mais configuration persistante erronée.

#### 104.5 — Permissions
Référence:
- shared-dropbox.

Compléments:
- répertoire partagé avec SGID/sticky/umask;
- audit d'un SUID/SGID injustifié et comportement multi-utilisateur.

#### 104.6 — Liens
- release déplacée avec symlink cassé;
- démontrer la différence de comportement entre hard link et symlink après renommage/suppression.

#### 104.7 — FHS
- fichier de configuration/log/data placé au mauvais endroit;
- retrouver les ressources d'une application en utilisant FHS et outils de recherche.

## 14. Portfolio cible — Exam 102

### Topic 105 — shells et scripts: 3 à 4 scénarios

#### 105.1 — Personnalisation de l'environnement
- environnement de login/interactif incohérent entre utilisateurs;
- alias, fonction, PATH et fichiers de startup à remettre en ordre.

#### 105.2 — Scripts shell
- script de maintenance qui traite mal arguments, codes retour et boucles;
- automatisation de rotation/rapport à rendre fiable et idempotente.

### Topic 106 — interfaces utilisateur: 3 à 5 scénarios

#### 106.1 — X11 et Wayland
- session graphique qui ne démarre pas avec variables/display/session incohérents;
- identifier ce qui relève de X11, Wayland ou du display manager.

#### 106.2 — Bureaux et accès distant
- choisir/configurer une méthode adaptée entre X forwarding, VNC, RDP/SPICE selon le scénario;
- diagnostic d'une session distante qui ne présente pas le bon environnement.

#### 106.3 — Accessibilité
- poste à adapter pour un profil utilisateur avec contraintes de vision, clavier ou pointage;
- vérifier une configuration réellement appliquée lorsqu'une stack graphique de test est disponible.

### Topic 107 — administration système: 5 à 6 scénarios

#### 107.1 — Utilisateurs et groupes
- onboarding d'un utilisateur avec groupe, home, shell, squelette et permissions;
- offboarding sans supprimer les données devant être conservées;
- compte système versus compte humain.

#### 107.2 — Planification
- tâche cron qui ne s'exécute pas dans son environnement non interactif;
- autorisations cron/at;
- tâche ponctuelle versus récurrente.

#### 107.3 — Locales, encodages, timezone
- application affichant du texte corrompu à cause d'un mauvais locale/encoding;
- timestamps incohérents entre timezone système et application.

### Topic 108 — services système: 5 à 6 scénarios

#### 108.1 — Heure
- dérive d'horloge qui casse corrélation de logs/authentification;
- configurer et vérifier une source de synchronisation.

#### 108.2 — Logs
- incident absent du fichier attendu à cause de journald/rsyslog/configuration;
- rotation cassée provoquant croissance disque ou perte prématurée.

#### 108.3 — Messagerie locale
- notification locale non distribuée;
- file d'attente ou alias local à diagnostiquer sans dépendre d'Internet.

#### 108.4 — CUPS
- job bloqué dans une file d'impression;
- imprimante/file incorrecte, annulation et diagnostic du spooler.

### Topic 109 — réseau: 6 à 8 scénarios

#### 109.1 — TCP/IP
- deux machines ne communiquent pas à cause d'adresse/masque/gateway;
- déterminer réseau, broadcast, ports et connexions à partir d'un incident.

#### 109.2 — Configuration persistante
- réseau réparé manuellement mais cassé après reboot;
- interface/gateway/DNS à rendre persistants selon la distribution de la VM.

#### 109.3 — Diagnostic
- service joignable localement mais pas depuis une autre machine;
- route, interface, socket ou filtrage à isoler méthodiquement.

#### 109.4 — DNS
- IP joignable mais nom non résolu;
- ordre de résolution /etc/hosts, NSS et DNS;
- réponse DNS correcte mais mauvais resolver local.

Les labs réseau doivent privilégier le backend multi-machine isolé lorsqu'il apporte une vraie compétence de diagnostic.

### Topic 110 — sécurité: 5 à 7 scénarios

#### 110.1 — Audit et tâches de sécurité
- comptes/permissions anormaux à auditer;
- recherche de fichiers à privilèges, processus/utilisateurs ou ressources à risque;
- politique de mots de passe/expiration dans une sandbox adaptée.

#### 110.2 — Durcissement
- service inutile exposé;
- socket activation/xinetd ou service legacy à limiter/désactiver;
- shell de compte de service et surface réseau à réduire.

#### 110.3 — SSH, GPG et cryptographie
- accès SSH cassé par permissions/known_hosts/authorized_keys;
- déployer une clé sans mot de passe transmis en clair;
- vérifier une signature GPG et détecter un artefact modifié.

## 15. Migration incrémentale

Les 618 micro-labs ne doivent pas être supprimés en une seule fois.

Pour chaque concept:
1. le fallback actuel reste disponible tant qu'aucun scénario accepté ne le couvre;
2. lorsqu'un scénario accepté couvre le concept, le scheduler préfère exclusivement le scénario;
3. l'audit marque le concept "scenario-covered";
4. le fallback peut rester techniquement présent mais n'est plus proposé;
5. lorsqu'un examen entier est couvert par des scénarios, ses fallbacks sont désactivés globalement;
6. après couverture complète 101 + 102 et période de stabilité, le générateur de micro-labs est supprimé.

Cette stratégie garantit zéro régression de couverture pendant la migration.

## 16. Phase A — contrat et infrastructure

Objectif: préparer la migration sans modifier brutalement l'expérience utilisateur.

Travail:
- formaliser le contrat scénario dans les schemas/docs;
- créer scenario-coverage.json;
- créer audit_scenario_coverage.py;
- distinguer fallback généré et scénario state-based dans l'evidence;
- permettre au scheduler de préférer un scénario accepté à un fallback;
- ajouter les tests positif/négatif/reset/reference solution;
- ajouter un compteur de couverture scénario dans la validation foundation.

Critère de sortie:
- l'état actuel passe encore;
- aucun concept ne perd sa pratique;
- la CI sait mesurer précisément la migration.

## 17. Phase B — pilote 103.1

103.1 est le pilote parce que shell-environment-repair fournit déjà le niveau de qualité recherché.

Objectif:
- couvrir les 12 concepts de 103.1 uniquement avec environ 4 à 5 scénarios réalistes;
- retirer tous les fallbacks 103.1 de la progression;
- tester le nouveau scheduler et la nouvelle evidence.

Scénarios de départ:
1. shell-environment-repair;
2. transfer-shell-handoff remanié si nécessaire;
3. résolution/identification de commande entre plusieurs utilisateurs;
4. quoting, expansion et séquences dans une tâche réelle;
5. éventuellement un scénario historique/documentation si la couverture restante le nécessite.

Critères de sortie:
- 12/12 concepts couverts;
- zéro fallback 103.1 proposé;
- tous les scénarios ont setup, reference solution, negative test et reset;
- review manuelle complète de chaque scénario;
- aucun brief ne révèle la procédure;
- chaque check prouve un comportement ou un état pertinent.

## 18. Phase C — migration de l'Exam 101

Ordre recommandé:

1. 103.5 et 104.5
   - plusieurs scénarios de référence existent déjà;
   - sert à stabiliser les patterns process/PTY et permissions.

2. reste du topic 103
   - haute densité de compétences directement praticables;
   - permet d'obtenir rapidement de nombreux scénarios Podman robustes.

3. topic 104
   - introduit davantage de KVM/filesystems;
   - renforce les checks d'état persistants.

4. topic 102
   - bootloader, packages, libraries, VM/cloud;
   - mélange Podman/KVM.

5. topic 101
   - architecture/boot/systemd;
   - finalise les scénarios système complets.

Critère de sortie Exam 101:
- 162/162 concepts scenario-covered;
- aucun fallback Exam 101 dans la progression;
- environ 27 à 35 scénarios acceptés, sauf justification documentée;
- foundation + Podman + KVM acceptance verts;
- audit pédagogique manuel complet;
- une session normale ne présente aucun exercice de preuve artificielle.

## 19. Phase D — review pédagogique Exam 101

Avant de passer à 102, jouer chaque scénario depuis un environnement propre comme un apprenant.

Pour chaque lab, vérifier:
- brief compréhensible sans connaissance du checker;
- diagnostic nécessaire mais pas obscur;
- niveau cohérent avec les cours précédents;
- plusieurs solutions plausibles;
- hints réellement progressifs;
- timeout suffisant;
- pas de dépendance à une connexion externe;
- :check explique précisément ce qui reste incorrect sans révéler toute la solution;
- reset fiable;
- debrief utile et court.

Mesures à enregistrer:
- temps de résolution attendu;
- nombre d'hints utilisés lors du test;
- concepts réellement exercés;
- commandes indispensables versus chemins alternatifs;
- bugs/points d'UX observés.

## 20. Phase E — migration de l'Exam 102

Appliquer le même processus aux topics 105 à 110.

Ordre recommandé:
1. 105 — shell/scripts;
2. 107 — utilisateurs, tâches, locales;
3. 108 — services;
4. 109 — réseau/multi-machine;
5. 110 — sécurité;
6. 106 — graphique/accessibilité, qui demande des environnements de test plus spécifiques.

Critère de sortie:
- 147/147 concepts scenario-covered;
- aucun fallback Exam 102 dans la progression;
- couverture state-based ou décisionnelle justifiée pour chaque concept;
- scénarios réseau et SSH réellement multi-contexte lorsque nécessaire;
- aucune dépendance Internet indispensable.

## 21. Phase F — retrait du générateur

Une fois les deux examens migrés et stabilisés:
- désactiver définitivement la génération des standalone labs;
- supprimer les 618 définitions générées de la surface utilisateur;
- supprimer le code de génération devenu mort;
- supprimer les tests spécifiques au fallback;
- conserver uniquement les outils de génération utiles à l'audit ou au développement si nécessaire;
- mettre à jour ROADMAP, README et documentation contributeur.

La suppression ne doit intervenir qu'après vérification que scenario-coverage affiche 309/309 concepts couverts.

## 22. Critères d'acceptation globaux

La migration est terminée lorsque:

### Couverture
- 309/309 concepts actifs ont une pratique scénarisée;
- aucun concept ne dépend d'un fallback générique;
- chaque concept est relié à une preuve explicite.

### Qualité
- aucun scénario accepté ne peut passer juste après setup;
- toutes les reference solutions passent;
- les checks observent l'état/comportement plutôt que l'historique de commandes;
- les briefs ne contiennent pas de procédure cachée;
- les hints suivent les quatre niveaux.

### Produit
- la progression sait grouper plusieurs concepts dans un même scénario;
- les evidences historiques restent compatibles;
- les scénarios sont sélectionnés de façon cohérente avec les concepts vus;
- les labs s'ouvrent et se reprennent sans dégrader l'UX du terminal.

### Sécurité
- isolation rootless/VM conservée;
- network deny-by-default;
- ressources plafonnées;
- pas de commandes destructrices sur l'hôte;
- snapshots/reset fiables pour les scénarios KVM.

### Maintenance
- matrice de couverture machine-readable;
- CI anti-régression;
- documentation contributeur;
- convention claire pour ajouter ou modifier un scénario.

## 23. Definition of Done d'un scénario

Un scénario est "accepted" uniquement si toutes les cases suivantes sont vraies:

- [ ] situation plausible et cohérente;
- [ ] concepts explicitement listés;
- [ ] brief sans solution implicite;
- [ ] setup déterministe;
- [ ] état initial réellement problématique;
- [ ] au moins une investigation utile nécessaire;
- [ ] checks indépendants de la commande exacte;
- [ ] mapping check -> concept;
- [ ] reference solution;
- [ ] reference solution verte;
- [ ] test négatif vert;
- [ ] reset testé;
- [ ] quatre niveaux d'hints;
- [ ] debrief;
- [ ] ressources/timeout adaptés;
- [ ] isolation vérifiée;
- [ ] review manuelle effectuée;
- [ ] matrice scenario-coverage mise à jour.

## 24. Definition of Done d'un objectif

Un objectif est "scenario-migrated" lorsque:
- tous ses concepts actifs sont couverts par au moins un scénario accepted;
- aucun fallback de cet objectif n'est proposé par le scheduler;
- les labs couvrent les commandes/fichiers/termes réellement examinables;
- l'audit ne signale aucun concept uniquement théorique sans justification;
- la progression course -> quiz -> scénario a été testée.

## 25. Definition of Done d'un examen

Un examen est "scenario-complete" lorsque:
- tous ses objectifs sont scenario-migrated;
- aucun fallback généré n'est visible;
- la CI complète passe;
- les scénarios KVM requis ont été testés sur un hôte réel;
- une review manuelle du parcours complet n'a trouvé aucun blocage majeur;
- les métriques de couverture sont archivées dans la documentation de l'examen.

## 26. Règles pour les futures contributions

Après cette migration, un nouveau concept ne doit pas automatiquement créer un nouveau lab.

Le contributeur doit d'abord répondre:
1. ce concept entre-t-il naturellement dans un scénario existant ?
2. faut-il enrichir les checks d'un scénario existant ?
3. une variante de transfert suffit-elle ?
4. un nouveau scénario apporte-t-il réellement une situation différente ?

Créer un nouveau scénario seulement lorsque la réponse est oui à la quatrième question ou lorsque les trois premières options dégradent la clarté pédagogique.

## 27. Premier jalon

Le premier jalon concret est:

> **103.1 scenario-complete: 12/12 concepts couverts par environ 4 à 5 vrais scénarios, zéro fallback généré visible, tous les checks state-based ou behavior-based, CI et review manuelle vertes.**

Ce jalon fixe le standard avant industrialisation sur le reste de l'Exam 101.
