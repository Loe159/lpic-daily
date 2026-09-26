# Schemas

Les contrats de contenu utilisent **JSON Schema Draft 2020-12**.

## Schemas

- `common.schema.json`: IDs, labels et types partagés.
- `objective-graph.schema.json`: graphe de prérequis des 42 objectifs.
- `concept.schema.json`: unité stable de maîtrise.
- `lesson.schema.json`: contenu explicatif progressif.
- `question.schema.json`: assessment déterministe hors ligne.
- `lab.schema.json`: environnement/checks sécurisés.
- `hint.schema.json`: hints progressifs.
- `mastery-evidence.schema.json`: événement append-only de compétence.
- `achievement.schema.json`: gamification sans effet sur mastery.

Voir `docs/CONTENT_MODEL.md`.

## Versioning

Chaque instance de contenu possède `schema_version`. Les schemas suivent SemVer. Les migrations seront implémentées dans l'application avant toute rupture de format.

## Validation actuelle

Les scripts foundation valident structure du curriculum, graphe, concept inventory et références. La validation JSON Schema complète sera intégrée au loader Go Phase 1 et aux tests CI; aucun contenu invalide ne doit être chargé en mode permissif.
