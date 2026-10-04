# LPIC Daily

> Un environnement d’apprentissage Linux **terminal-first**, **local-first** et **offline-first** pour préparer LPIC-1 en pratiquant réellement l’administration système.

LPIC Daily transforme le programme LPIC-1 v5.0 (101-500 + 102-500) en une boucle quotidienne : micro-cours, rappel actif, questions déterministes, labs jetables, progression adaptative et notifications desktop.

Le projet privilégie les compétences transférables plutôt que la mémorisation de commandes : les labs évaluent l’**état final observable**, pas une séquence de commandes imposée.

## État du projet

- **Phase 1 — boucle d’apprentissage complète : validée**
- **Phase 2 — labs VM libvirt/QEMU/KVM : validée sur hôte réel le 4 octobre 2026**
- **Phase 3 — couverture Exam 101 : en cours**
- Objectif 103.4 déjà intégré avec micro-leçons et questions quotidiennes déterministes

Le scope machine-readable couvre actuellement **42 objectifs LPIC-1**, **295 concepts stables** et l’intégralité des **23 objectifs / 153 concepts Exam 101** comme cible de Phase 3.

## Pourquoi LPIC Daily

- **Pratique réelle, environnement jetable** : Podman rootless pour les labs légers, KVM/libvirt pour boot, stockage et scénarios full-system.
- **Sécurité fail-closed** : aucun fallback vers l’exécution directe sur l’hôte.
- **Progression adaptative** : prérequis, maîtrise, révisions dues et poids LPIC influencent la séance.
- **Maîtrise fondée sur des preuves** : lecture, reconnaissance, rappel, pratique guidée, pratique autonome et transfer sont distingués.
- **French-first** : explications en français, vocabulaire Linux conservé en anglais lorsqu’il est naturel.
- **Local-first** : SQLite local, fonctionnement cœur sans cloud, compte ni IA.
- **Gamification séparée** : XP, streaks et achievements motivent sans falsifier la maîtrise.
- **Traçabilité LPIC** : chaque activité reste reliée aux objectifs officiels et à des concept IDs stables.

## Démarrage rapide

### Prérequis de développement

- Linux
- Go **1.27+**
- Python 3
- Podman pour les labs conteneur
- KVM/libvirt + QEMU pour les labs VM

### Build

```bash
git clone https://github.com/Loe159/lpic-daily.git
cd lpic-daily
go build -o lpic ./cmd/lpic
./lpic
```

Au premier lancement interactif, LPIC Daily configure automatiquement ce qui peut l’être sans privilèges :

- copie du binaire utilisateur dans `~/.local/bin/lpic`;
- timer systemd utilisateur pour les notifications quotidiennes;
- desktop entry;
- socket Podman rootless lorsqu’il est disponible;
- détection du terminal utilisé par l’action de notification.

Les opérations privilégiées ou réseau importantes restent explicites et demandent confirmation.

Pour préparer l’environnement à l’avance :

```bash
lpic install
```

Pour différer KVM/libvirt :

```bash
lpic install --no-vm
```

Pour une installation non interactive de développement :

```bash
lpic install --yes
```

## Utilisation

```bash
lpic                    # dashboard TUI
lpic today              # séance du jour
lpic today --quick      # séance courte
lpic assess             # évaluation initiale
lpic doctor             # diagnostic de l’environnement
lpic lab list           # labs disponibles
lpic lab run <lab-id>   # lancer un lab
lpic notify --force     # tester la notification
lpic validate           # valider le contenu embarqué
```

Dans un lab :

```text
:shell    shell PTY persistant dans la sandbox
:check    évaluer l’état final
:hint     afficher l’indice suivant
:reset    recréer l’état jetable
:quit     détruire le lab et quitter
```

Les commandes apprenant sont exécutées **dans la sandbox uniquement**.

## Isolation

### Podman rootless

Utilisé pour shell, fichiers, processus et exercices ne nécessitant pas une machine complète.

Principes :

- rootless uniquement;
- rootfs en lecture seule lorsque possible;
- chemins écrivable explicitement bornés;
- aucune montage arbitraire de l’hôte;
- réseau désactivé par défaut;
- limites CPU / RAM / PID / durée;
- image résolue vers une identité immuable avant création.

### libvirt / QEMU / KVM

Utilisé lorsque l’objectif dépend du kernel, du bootloader, du stockage bloc ou d’une machine complète.

Principes :

- processus LPIC Daily non-root;
- `qemu:///system` local uniquement;
- bases QCOW2 vérifiées et immuables;
- overlays jetables par exécution;
- aucun passthrough PCI/USB arbitraire;
- réseaux isolés sans forwarding Internet/LAN;
- ownership des ressources libvirt vérifié;
- nettoyage et reaping des ressources abandonnées.

Voir `docs/SECURITY.md` et `docs/THREAT_MODEL.md`.

## Architecture

```text
TUI / CLI
   |
   +-- curriculum ------ machine-readable LPIC scope
   +-- content --------- lessons / questions
   +-- learning -------- evidence / mastery / scheduler
   +-- progress -------- SQLite
   +-- lab ------------- orchestration
       |
       +-- runner/podman
       +-- runner/libvirt
       +-- checker
```

Principaux répertoires :

```text
cmd/lpic/              CLI et orchestration
internal/learning/     evidence, mastery, scheduler
internal/progress/     persistence SQLite
internal/runner/       contrats de sandbox + backends
internal/checker/      grading par état observable
internal/lab/          chargement et orchestration des labs
content/               micro-leçons et questions
curriculum/            graphe LPIC et concept IDs
labs/                  labs et fixtures
schemas/               contrats JSON
docs/                  architecture, sécurité, produit, ADR
scripts/               validations et tooling
```

## Validation

Validation standard :

```bash
python3 scripts/validate_foundation.py
go mod tidy && git diff --exit-code -- go.mod go.sum
test -z "$(find . -name '*.go' -type f -print0 | xargs -0 gofmt -l)"
go test ./...
go vet ./...
go run ./cmd/lpic validate
```

Acceptance KVM sur un hôte compatible :

```bash
LPIC_DAILY_RUN_KVM_INTEGRATION=1 scripts/run_phase2_acceptance.sh
```

Audit de progression Exam 101 :

```bash
python3 scripts/audit_phase3_coverage.py --check
```

Gate de couverture théorique Phase 3 :

```bash
python3 scripts/audit_phase3_coverage.py --require-complete
```

## Contribuer

Avant une modification importante :

1. lire `AGENTS.md`;
2. lire les ADR et documents du domaine modifié;
3. conserver les invariants de sécurité et la traçabilité objective/concept;
4. ajouter ou mettre à jour les tests;
5. exécuter les validations pertinentes;
6. mettre à jour la documentation si le comportement change.

Les règles détaillées de contribution assistée par agents sont dans `docs/CONTRIBUTING_WITH_AGENTS.md`.

## Roadmap

- Phase 3 : terminer Exam 101 (topics 101–104)
- Phase 4 : couvrir Exam 102 (topics 105–110)
- Phase 5 : renforcer interleaving, transfer, assessments et explicabilité
- Phase 6 : simulation d’examen, packaging, update/signing et hardening final

Voir `docs/plan/ROADMAP.md`.

## Licences

- Code : **Apache-2.0** — voir `LICENSE`
- Contenu éducatif et documentation originale : **CC BY 4.0** — voir `LICENSE-CONTENT.md`

LPIC Daily est un projet indépendant. Les contenus pédagogiques sont originaux et ne reproduisent pas les supports propriétaires de LPI.
