# Revue pédagogique des labs LPIC-101 — 14 octobre 2026

## Portée

Audit statique des **70 scénarios** du programme 101 (dont un scénario de compatibilité), avec comparaison des `brief_fr`, `success_criteria_fr`, `concept_ids`, contrôles et, pour les cas à risque, des scripts de préparation et solutions.

À la suite d'un retour utilisateur sur une consigne trop directive, l'audit applique désormais explicitement le principe **mission / résultat attendu / contraintes, pas une recette**. Les commandes, variables de configuration et séquences d'exécution doivent être découvertes. Les indices sont progressifs, la solution de référence restant réservée aux tests.

## Résultat

- **30 labs corrigés**, sur 70 relus. Les 40 autres conservent leurs descriptions lorsque le contexte, les contraintes et le résultat suffisent sans donner la procédure.
- **103.1/transfer-shell-handoff** : la consigne exprime les capacités de l'historique (1 500 commandes en session, 3 000 sur disque, doublons évités) au lieu de livrer directement les variables `HISTSIZE`, `HISTFILESIZE` ou `HISTCONTROL`. Elle décrit aussi les besoins opérationnels des outils et du rapport sans préciser les changements à faire dans `.bashrc`.
- **103.1/history-handoff-recovery** et **103.1/shell-environment-repair** : tailles d'historique exprimées comme résultats, sans prescriptions de variables dans le brief.
- **103.5** : suppression des commandes de contrôle de jobs (`jobs`, `bg`, `nohup`, `tmux`) dans les missions ; suspension, reprise, continuité et session détachée restent les objectifs vérifiables.
- **104.5** : permissions et isolation exprimées en termes de collaboration, héritage de groupe, suppression protégée et absence d'accès extérieur plutôt qu'en octal et réglages de `umask`.
- **101.1** et **101.2** : les consignes ne révèlent plus d'emblée la cause ou le composant Linux à réparer lorsque le diagnostic fait partie du concept.
- **103.2/incident-report-rebuild** : anciennes attentes fixes `web_02` trompeuses ; les hôtes sont randomisés. Le critère repose désormais sur les vrais fichiers de la session.
- **103.4/bulk-argument-dispatch** : ancienne taille de fichier constante `beta.txt:11` trompeuse ; le résultat attendu est dynamique.
- **103.1/transfer-shell-handoff** : suppression du fichier artificiel `handoff-ready` et de son contrôle, y compris les indices et la solution de référence.
- Deux titres génériques en anglais traduits (entrée GRUB alternative et provisionnement NoCloud).

## Périmètre des modifications

| Objectif | Labs présents | Labs corrigés |
| --- | ---: | ---: |
| 101.1 | 6 | 3 |
| 101.2 | 5 | 1 |
| 101.3 | 8 | 1 |
| 102.1 | 3 | 0 |
| 102.2 | 5 | 3 |
| 102.3 | 2 | 0 |
| 102.4 | 3 | 0 |
| 102.5 | 4 | 0 |
| 102.6 | 4 | 1 |
| 103.1 | 6 | 6 |
| 103.2 | 2 | 2 |
| 103.3 | 2 | 2 |
| 103.4 | 2 | 2 |
| 103.5 | 2 | 2 |
| 103.6 | 1 | 1 |
| 103.7 | 1 | 1 |
| 103.8 | 1 | 1 |
| 104.1 | 4 | 1 |
| 104.2 | 2 | 0 |
| 104.3 | 3 | 0 |
| 104.5 | 2 | 2 |
| 104.6 | 1 | 0 |
| 104.7 | 1 | 1 |
| **Total** | **70** | **30** |

## Barème de clarté et pertinence

1. **Contexte** : problème plausible, impacts et priorité clairs ; ne pas divulguer d'emblée le diagnostic à découvrir.
2. **Attendu** : critères mesurables, quantités, chemins et formats de sortie uniquement lorsqu'ils font réellement partie du besoin.
3. **Autonomie** : ne pas prescrire commandes, options, fichiers de configuration à éditer, valeurs `KEY=value`, permissions octales ou séquences de manipulation si ce sont justement les connaissances évaluées.
4. **Concept** : chaque action contribue directement à une notion LPIC déclarée, plutôt qu'à un fichier de preuve artificiel.
5. **Contrôles** : la formulation doit correspondre aux invariants effectivement vérifiés ; n'annoncer aucune valeur constante pour des entrées aléatoires.
6. **Indices** : observation → diagnostic → mécanisme → éventuelle solution. Les commandes précises peuvent figurer dans les indices avancés, pas dans la consigne initiale.
7. **Lisibilité** : idéalement un bref paragraphe pour le problème et ses contraintes ; les détails exacts des preuves sont dans les critères.

Les règles de rédaction obligatoires sont ajoutées à `labs/AGENTS.md` pour les nouvelles contributions.

## Vérification restante

Cette revue **ne certifie pas l'exécution des 70 labs**. Les JSON modifiés restent structurellement lisibles ; les contrôles Podman et KVM doivent être exécutés avant validation complète. Dans plusieurs scénarios, les contrôles d'état démontrent la validité du résultat final mais non la méthode réellement employée ; ne pas revendiquer une preuve de l'utilisation de `xargs`, `tee` ou `find` sans instrumentation adaptée. Enfin, valider en TUI que les missions et critères restent visibles sans troncature.
