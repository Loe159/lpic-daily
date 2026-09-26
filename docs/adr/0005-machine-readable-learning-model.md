# ADR 0005 — Graphe et contenu machine-readable

Status: **Accepted — 2026-09-26**

## Decision

Adopter:

- un graphe objective-level avec hard/recommended prerequisites;
- des concept IDs stables comme granularité primaire de maîtrise;
- des learning items versionnés par JSON Schema;
- un journal append-only de mastery evidence;
- une séparation structurelle entre mastery et gamification.

## Rationale

Les objectifs LPIC sont assez fins pour garantir la couverture d'examen mais trop grossiers pour un scheduler adaptatif. Inversement, figer immédiatement un graphe détaillé entre ~300 concepts serait spéculatif avant d'avoir écrit les activités.

Le modèle à deux niveaux garde donc:
- objective graph: stable et complet dès maintenant;
- concept inventory: stable IDs;
- concept-to-concept edges: ajoutés seulement lorsque le contenu réel démontre leur utilité.

Le journal d'évidence permet de changer l'algorithme de maîtrise sans perdre l'historique ni l'explicabilité.

## Consequences

- Toute nouvelle lesson/question/lab référence des concept IDs et objective IDs.
- Les schemas sont versionnés et validés avant chargement.
- Un objective score est dérivé des concepts.
- XP/streaks/achievements ne modifient jamais directement le mastery.
- Le scheduler doit expliquer les prerequisites qui ont rendu un concept éligible ou non.
