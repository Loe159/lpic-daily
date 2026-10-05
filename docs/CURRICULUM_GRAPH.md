# Graphe pédagogique LPIC-1

Source machine-readable: `curriculum/lpic-1-v5/prerequisites.json`.

## But

Les numéros LPIC décrivent le syllabus, pas un ordre pédagogique optimal. Le graphe permet au scheduler de choisir du nouveau contenu dont les fondations sont disponibles, puis d'intercaler révisions et objectifs connexes.

Le graphe couvre les **42 objectifs actifs LPIC-1 v5.0**. Les notions internes sont décomposées en **309 concept IDs stables** dans `curriculum/lpic-1-v5/concepts.json`.

## Deux types d'arêtes

### Hard prerequisite

Un objectif ne devrait normalement pas être introduit comme nouveau contenu autonome avant que ses hard prerequisites aient un minimum d'évidence.

Exemples:

- `103.1 -> 103.5`: comprendre le shell avant jobs/processus.
- `104.1 -> 104.2`: comprendre/créer un filesystem avant de le réparer.
- `101.2 + 102.1 + 104.3 -> 102.2`: réparer GRUB suppose boot + stockage + montage.

Ce n'est pas un verrou absolu: un assessment initial ou une preuve importée peut satisfaire le prerequisite sans forcer le cours correspondant.

### Recommended prerequisite

C'est un signal d'ordonnancement, pas un blocage.

Exemple: `103.2` (filtres texte) est recommandé avant d'approfondir certaines regex, mais `103.7` peut être introduit sans attendre tout `103.2`.

## Point d'entrée

`103.1 — Travail en ligne de commande` est le point d'entrée recommandé car il débloque directement une grande partie des labs.

D'autres racines restent volontairement indépendantes:

- `101.1` matériel;
- `101.2` modèle conceptuel du boot;
- `106.3` accessibilité;
- `109.1` TCP/IP.

Cela permet de varier les séances et d'éviter une longue séquence uniquement shell au début.

## Utilisation par le scheduler

Le scheduler construit une frontier d'objectifs/concepts:

1. retirer le contenu déjà maîtrisé sauf s'il est dû en révision;
2. écarter le nouveau contenu dont les hard prerequisites ne sont pas satisfaits;
3. pondérer le reste par poids LPIC, faiblesse, recency, recommended prerequisites, diversité et durée;
4. expliquer la sélection à l'utilisateur;
5. ne jamais confondre ordre d'apprentissage et pourcentage de couverture LPIC.

## Concepts

Un objective est trop grossier pour piloter la maîtrise. Chaque entrée de `concepts.json` possède:

- un ID immuable;
- son objective LPIC parent;
- son libellé français;
- un ordre pédagogique local indicatif;
- les hard prerequisites hérités de l'objectif.

Les dépendances concept-to-concept fines seront ajoutées seulement quand le contenu réel le justifie. On évite ainsi d'inventer 295 relations spéculatives avant les lessons/labs.

## Invariants

- Le graphe hard est acyclique.
- Le graphe hard + recommended est aussi acyclique afin de conserver une lecture pédagogique simple.
- Chaque objectif actif a exactement un node.
- Les références pointent uniquement vers des objectifs actifs.
- Chaque objective dispose d'au moins un concept stable.
- Un vertical slice doit être fermé sur ses hard prerequisites ou documenter explicitement l'évidence externe qui les satisfait.
