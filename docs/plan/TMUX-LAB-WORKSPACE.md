# Plan d'implémentation — espace lab tmux (maquette B)

Statut : **première implémentation dans la PR #56 (2026-10-19), tests Go/CI et validation manuelle tmux/PTY requis avant acceptation.** La présente note conserve le périmètre cible et les étapes de validation ; certains perfectionnements sont encore à valider sur un hôte interactif.

## Objectif / résultat utilisateur

Depuis la TUI LPIC Daily, sélectionner un lab ouvre une **unique session de travail tmux**, dans **le terminal existant** :

- **gauche, 40 %** : écran LPIC Daily immobile composé de (1) consigne + objectifs + indices progressifs, et (2) résultats du dernier `:check` dans un cadre inférieur indépendant ;
- **droite, 60 %** : terminal Linux réellement interactif du lab (Podman ou console/VM suivant backend), dont la sortie est réservée au travail Linux ;
- bordure et barre de statut avec titre, backend, raccourcis ; pas de nouvelle fenêtre ni de texte de check dans l'historique shell.

Conserver les quatre indices validés et leur état SQLite. La PR #53 a été fermée sans fusion : elle ne doit **pas** être reprise en bloc. Le projet doit garder le comportement existant tant que le nouveau parcours n'est pas accepté.

## Constats à partir du code existant

- `cmd/lpic/main.go:runDashboardLab` ouvre actuellement une nouvelle fenêtre via `desktop.LaunchLabWithCheckSocket`; `waitForDashboardLabCompletion` attend dans le parent et déclenche `:check`.
- `runInteractiveLabWithBackend` est propriétaire de la sandbox et du cycle de vie. `runPersistentShell` passe un flux PTY brut du guest à son terminal, avec transfert de SIGWINCH, gestion spéciale Ctrl-Z, `:check`, `:hint`, `:quit`.
- `cmd/lpic/check_bridge.go` implémente déjà un canal Unix parent ↔ child pour les checks, non pour les indices/événements de rendu.
- `internal/desktop/notify.go` fournit le lanceur actuel et son fallback.
- `docs/adr/0013-interactive-podman-pty.md` interdit les shells Linux du lab sur l'hôte : Podman exécute les commandes dans une sandbox à travers son API. **Ne pas changer cette limite.**
- L'image Fedora inclut tmux *dans le guest* pour des labs comme 103.5. Cette installation est distincte du **tmux hôte**, qui devra être détecté/installer optionnellement.

## Architecture décidée

```text
Terminal local existant (host)
  └─ tmux local, socket dédié LPIC, session lpic-lab-<random-id>
       ├─ panneau gauche : lpic lab panel --session <opaque-id>
       │    ├─ haut : mission + objectifs + indices divulgués
       │    └─ bas : derniers résultats détaillés du check (indépendant)
       └─ panneau droit : lpic lab run <id> --workspace-session <opaque-id>
            └─ backend Runner API → Podman PTY / libvirt console, jamais un shell hôte

Orchestrateur parent/child : canal Unix local déjà éprouvé (étendu),
état sauvegardé atomiquement pour affichage et reconnexion.
```

Le panneau gauche est une application Go TUI en mode **lecture/navigation** ; le panneau droit reste le seul à recevoir des commandes Bash/vi/nano/fg/bg. Le panneau gauche n'exécute aucune commande de lab. Séparer clairement les processus d'interface et le moteur du lab (possibilité de mutualiser un seul processus contrôleur avec composants distincts lors de l'implémentation si tests le justifient).

**Approche recommandée en étapes, pas de réécriture massive de la TUI** : ajouter le gestionnaire de sessions tmux host derrière une interface testable, réutiliser `runInteractiveLabWithBackend` comme worker ; ajouter une surface gauche événementielle indépendante et garantir qu'il n'y a qu'un seul processus autorisé à noter la réussite.

## Disposition / adaptativité

- >= 100 colonnes : gauche 40 % / droite 60 % au lancement ; redimensionnement manuel tmux autorisé ; conserver une largeur de shell utilisable.
- Hauteur du panneau gauche : espace haut flexible, **zone de résultats ancrée en bas** avec défilement et une synthèse « N/M OK ». Les indices révélés sont ajoutés sous la mission, non mélangés aux lignes du shell ; navigation indépendante (PageUp/Down ou flèches lorsque focus gauche).
- Entre 80 et 99 colonnes : largeur gauche réduite avec reflow de la consigne ; avertissement si contenu critique serait masqué ; mode une seule fenêtre zoomable.
- < 80 colonnes ou terminal non interactif : fallback existant, clair, non destructeur (mode classique ou bascule vers vue mono-panneau). Ne jamais tenter un split inutilisable.
- Terminal droit : support correct du resize PTY, Ctrl-Z, Ctrl-C, Home/End, nano, vim, commandes interactives, job control ; pas d'injection de séquences de contrôle provenant des checks.
- Préfixe tmux **hôte** distinct et annoncé ; ne pas voler `Ctrl+B` lorsque le lab enseigne tmux. Les tmux utilisés **dans** le guest restent accessibles (nested tmux). Prévoir `F6` ou commande de palette pour focus, et raccourci de zoom du shell vérifié en environnement utilisateur avant stabilisation.
- Barre d'aide non envahissante : « :hint  :check  :reset  :quit », et indication du panneau actif.

## Événements et modèle de session

Définir des messages *typés*, avec versions, bornes de taille et temps de réponse :
- `LabStarted` : lab ID, titre, objectif, backend, heure, état `starting|ready` ;
- `HintRevealed` : niveau 1–4, contenu Markdown **sanitisé**, impact evidence, horodatage ;
- `CheckStarted`, `CheckCompleted` : résultat par critère stable, libellé français, détails bornés, compteur ;
- `LabPassed`, `LabError`, `LabReset`, `LabEnded` et éventuellement `LabDetached`.

**Autorité unique** : processus du lab pour les checks et les indices ; écriture de SQLite centrale (ne pas reclassifier la maîtrise dans le panneau). En cas de requêtes simultanées, sérialisation/transaction de la divulgation des indices, pour éviter deux fois le niveau 1 ou des niveaux sautés. La reprise relit le niveau réel de divulgation SQLite.

Le panneau ne lit pas directement les fichiers du guest. Il ne reçoit que des données structurées de l'orchestrateur, bornées et assainies (`lpicui.SanitizeText`, rendu Markdown contrôlé). La sortie brute de Podman/libvirt n'arrive qu'au PTY du panneau droit.

Pour les tests/checks : conserver le moteur actuel, mais renvoyer une liste structurée de critères plutôt que ré-analyser des lignes terminal. Le dernier état reste affiché après les commandes ; rafraîchir uniquement après `:check`, pas à chaque frappe.

## Cycle de vie et sécurité

1. TUI sélectionne le lab ; contrôle des prérequis, tmux hôte et taille avant lancement.
2. Créer une session unique portant un ID opaque aléatoire et un socket tmux LPIC **privé 0700** (perms utilisateurs), démarrer le contrôleur puis le worker en argv structurés, sans shell-string non échappé ni titre/ID fourni par contenu externe injecté dans des commandes host.
3. Lancer la sandbox comme aujourd'hui. Le panneau gauche observe `starting`, puis `ready` ou une erreur diagnostiquable.
4. **Détacher ≠ quitter** : la session continue si le client tmux disparaît ; reattacher à la même session, récupérer les indices et le dernier check sans reset.
5. `:quit` explicite demande confirmation quand un lab est en cours, ferme la sandbox puis la session LPIC uniquement ; une sortie involontaire ne doit pas détruire la VM/container.
6. `:reset` préserve l'historique de divulgation comme aujourd'hui, recrée seulement la sandbox, remet les checks à « non exécuté ».
7. TUI doit pouvoir retrouver `lpic-lab-*` et afficher « Reprendre / Recommencer / Quitter » quand un lab actif existe. Restreindre la récupération aux sessions manifestement créées par LPIC et au même utilisateur.
8. Reaper explicite pour sessions orphelines avec états/PID propriétaire ou socket fermée ; éviter les nettoyages basés sur un simple `tmux kill-server`, qui pourrait toucher des sessions personnelles.
9. **Host tmux** et **guest tmux** isolés ; aucune propagation du socket host `TMUX`, Podman, SSH_AGENT ou home dans le guest. Pas d'escalade de privilèges.

## Découpage des changements

### Lot 1 — Infrastructure sans changement d'interface

- Nouveau `internal/desktop/tmux.go` ou `internal/labworkspace/tmux.go` : détection binaire host, `TmuxManager` injectable (exec argv), session socket namespace, create/split/attach/detach/kill ciblé, taille initiale, propre fonctionnement tmux hôte déjà imbriqué.
- Tests fake executor sur l'assemblage argv, l'absence d'interpolation dangereuse, réattachement, isolation des sockets, erreurs tmux, fallback.
- Ajouter une commande interne `lpic lab workspace ...` non exposée dans l'aide générale tant qu'elle n'est pas stabilisée.

### Lot 2 — Panneau gauche

- Nouveau renderer Go dédié (peut réutiliser primitives TUI), indépendant de l'historique CLI.
- Bloc mission/critères et indices révélés en haut ; bloc de vérification fixé en bas. Styles lisibles en monochrome avec symboles, couleurs optionnelles et détails explicites.
- Gestion du scroll local, focus, minimum de largeur/hauteur, récupération depuis un snapshot.

### Lot 3 — IPC et commandes

- Étendre le bridge de manière rétrocompatible à `hint`, `reset`, `quit`, `status` et événement `check` ; authentifier/limiter canal Unix au même user et aux opérations autorisées.
- Les actions `:hint` et `:check` saisies à droite sont interceptées côté hôte comme actuellement ; la sortie vient **uniquement dans le panneau gauche**, et le shell affiche au plus une confirmation brève (« Consulte le panneau gauche »).
- Le propriétaire du lab enregistre dans SQLite avant d'envoyer `HintRevealed`. Les indices relus ne doivent ni élever le niveau, ni compter comme nouvelles divulgations.

### Lot 4 — Transition depuis la TUI

- Remplacer le lancement de fenêtre dans `runDashboardLab` par l'attachement tmux dans le **terminal actuel**, sous feature flag lors des premiers tests.
- Préserver le chemin `lpic lab run ...` historique pour automation/CLI/non-TTY et fallback ; gestion robuste des ressources après interruption.
- Documenter et rendre tmux hôte optionnel via `lpic doctor` et installateur, avec diagnostic « tmux manquant » plutôt que crash.

### Lot 5 — Acceptation

- Podman Fedora : 103.1, 103.4, 103.5, un lab nécessitant Ctrl-Z/jobs, un éditeur interactif nano/vim, :hint 1→4, checks consécutifs, reset, réussite.
- VM/KVM : console, redimensionnement, port série, Ctrl-] et redémarrage guest ; si split perturbe le workflow, fallback explicite.
- Tests réels : l'utilisateur reste AFK, ferme/rattache le terminal, vérifie que l'instance n'est ni détruite ni réinitialisée (régression signalée historiquement).
- Vérifier que les consignes ne sont jamais tronquées : scroll haut ; vérification toujours accessible bas ; sortie shell non parasitée.
- Cas : nested tmux et lab 103.5 (tmux guest), petites fenêtres, absence tmux, altération de socket, commande hostile, plusieurs labs, crash du contrôleur, Ctrl-C, détachement, check instantané.
- Mesurer le temps de `:check` : pas de shell interactif jetable par critère et pas de délai notable introduit par tmux.

## Critères d'acceptation produit

1. Un lab démarré via TUI se déroule dans la **même fenêtre terminal**, 40/60 ; la mission et les checks restent visibles.
2. Toutes les commandes et touches du shell sont fonctionnelles ; job control testé.
3. `:hint` ajoute un seul indice au panneau gauche, dans l'ordre, et l'impact maîtrise est identique au mode historique ; tous les indices déjà débloqués restent consultables.
4. `:check` affiche rapidement les résultats à gauche sans encombrer le shell ; états d'erreur détaillés et intelligibles.
5. Détacher/rattacher ne perd ni le shell ni la progression du lab ; `:quit` nettoie uniquement les ressources LPIC concernées.
6. Le fallback non tmux demeure utilisable ; pas de changement de privilèges ou de frontière sandbox.
7. Tests Go, foundation et conformance Podman/KVM applicables passent ; documentation de sécurité/architecture et tests mis à jour.
8. Aucun changement au contenu/scoring des labs ni à la matrice LPIC, sauf nécessité explicitement démontrée par les tests.

## Séquence de livraison suggérée

- **PR A** : infrastructure tmux + tests (aucun changement visible).
- **PR B** : panneau gauche + IPC + tests (activation manuelle).
- **PR C** : intégration TUI + reprise/fallback + tests d'intégration et documentation.
- Pas de merge de PR C avant validation visuelle sur des captures de taille réelle, ainsi que les tests d'éditeur, job control et détachement.

## Hors périmètre

Pas de refonte du curriculum, des règles de `:check` ni des scénarios. Pas de remplacement de Podman/libvirt par un shell hôte. Pas de dépendance à tmux **dans** le guest pour les panels : le split appartient uniquement au host.
