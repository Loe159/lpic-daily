# Modèle de contenu machine-readable

Les schémas sont sous `schemas/` et utilisent JSON Schema Draft 2020-12.

## Unités

### Concept

Un concept est la plus petite unité durable de suivi de maîtrise. Son ID ne change pas lorsque le wording pédagogique évolue.

Exemple:

`lpic1.103.5.processus-avant-arriere-plan`

Le niveau objective/topic est une agrégation de concepts, jamais l'unique source de maîtrise.

### Lesson

Une lesson explique un ou plusieurs concepts. `stage` indique le rôle:

- `introduce`
- `deepen`
- `apply`
- `transfer`
- `exam-review`

Lire une lesson produit au mieux de l'évidence d'exposition.

### Question

Une question doit être gradable hors ligne et de manière déterministe. Aucun LLM n'est nécessaire pour accepter/refuser une réponse.

Les stratégies initiales sont text exact, regex, choix d'IDs ou ordre d'IDs. Les questions libres dont la correction nécessiterait une interprétation sémantique ne font pas partie du contrat Phase 1.

### Lab

Un lab décrit l'activité pédagogique **et** son environnement exécutable.

La partie learner-facing contient au minimum:
- `title_fr`;
- `brief_fr`: scénario et objectif sans donner la solution;
- `success_criteria_fr`: comportements observables attendus;
- `debrief_fr`: explication révélée après validation/abandon.

La partie exécutable décrit environnement, ressources, setup **dans la sandbox**, checks structurés et hints.

Principes:
- backend explicite `podman` ou `libvirt`;
- un lab Podman utilise uniquement un profil de capacités prédéfini (`baseline`, `identity-files`, `metadata-db`, `package-root`, `process-lab`); un lab libvirt utilise `full-machine` et ne déclare pas de `writable_guest_paths`, car son état modifiable provient exclusivement des disques VM jetables;
- réseau `none` ou `isolated`, jamais Internet implicite;
- aucun host mount arbitraire dans le format;
- setup exécuté uniquement dans la sandbox;
- checks basés sur l'état final;
- reset `disposable`;
- référence solution réservée aux tests/authoring.

Les success criteria expliquent le résultat attendu mais ne doivent pas imposer une séquence de commandes. Les checkers restent la définition machine-readable du succès.

### Hint

Chaque lab possède exactement quatre hints, un par niveau 1–4. Le niveau 1 reste léger (`none` ou `minor`), les niveaux 2 et 3 sont `material` et réduisent une preuve pratique à `guided-practice`, et le niveau 4 révèle la solution (`solution-revealed`). Le contrat d'authoring et la projection de mastery utilisent ainsi la même frontière de preuve.

### Mastery evidence

La progression brute est un journal d'événements append-only. Une evidence enregistre:

- concept;
- activité source;
- type d'évidence;
- résultat;
- niveau de hint;
- révélation éventuelle de solution;
- distribution;
- tentative;
- date.

Le mastery score/état est une **projection recalculable** de ces événements. Ne stocker qu'un score mutable ferait perdre l'explicabilité et compliquerait les migrations d'algorithme.

### Achievement

Les achievements/XP sont explicitement `affects_mastery: false`. La gamification ne peut donc pas devenir accidentellement une preuve de compétence.

## IDs et versioning

- IDs de contenu: minuscules, stables et lisibles.
- Concept IDs publiés: immuables.
- `schema_version` suit SemVer.
- changement incompatible: major;
- ajout optionnel compatible: minor;
- clarification sans changement structurel: patch.

Une migration de schema ne doit jamais modifier silencieusement la signification d'une preuve historique.

## Labels pédagogiques

Les learning items utilisent:

- `lpic-required`;
- `lpic-legacy`;
- `modern-practice`.

Une même lesson peut couvrir plusieurs labels si elle compare explicitement approche examinable et pratique moderne.

## Sécurité

Les formats de contenu ne sont pas une API d'administration du host. Toute opération exécutable appartient à une sandbox assignée par le runner. Les schemas ne permettent pas de demander `--privileged`, un bind mount arbitraire ou un accès Internet public.
