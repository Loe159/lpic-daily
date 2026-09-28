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

Chaque instance de contenu possède `schema_version`. Les schemas suivent SemVer. Les migrations doivent être implémentées dans l'application avant toute rupture de format.

## Validation actuelle

Les scripts foundation valident la structure du curriculum, le graphe, l'inventaire des concepts, les références et la couverture machine-readable.

Les loaders Go valident le contenu authored contre les schemas Draft 2020-12 embarqués avant la validation sémantique ou toute exécution. Un document invalide ou un type runtime non supporté échoue explicitement; il n'existe pas de chargement permissif.

La CI exécute ces validations sur Ubuntu et valide également le contenu embarqué à travers le loader Go sur Ubuntu et Fedora.
