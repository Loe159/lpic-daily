# Disposition tmux B pour LPIC Daily

## Fonctionnement validé

Une fenêtre de terminal et deux panneaux : 40 % à gauche pour la mission, les indices et les vérifications ; 60 % à droite pour le shell Linux réel.

L'orchestrateur lance tmux sur l'hôte avec un socket indépendant nommé lpic-daily-ui. Le panneau gauche est un lecteur local, jamais une interface qui exécute des commandes apprenant. Le panneau droit utilise les runners Podman et KVM déjà présents. Les changements d'état sont transmis par un JSON atomique dans un dossier temporaire privé (0700) ; aucun flux PTY brut de la sandbox n'entre dans le panneau d'instructions.

## Commandes

- Dans le shell droit : :check met à jour les résultats à gauche ; :hint dévoile le prochain indice à gauche ; :quit termine le lab.
- Ctrl+G puis d : détacher tmux. Sélectionner à nouveau le même lab rattache la session active.
- Ctrl+G puis z : zoom. Ctrl+G puis flèche : sélectionner un panneau.
- Dans le panneau gauche : j et k font défiler les longues consignes, g retourne au début.
- Ctrl+B est réservé au shell et au tmux de l'environnement d'exercice.

## Repli et dépendances

La disponibilité de tmux sur l'hôte est facultative : si absent, si le terminal mesure moins de 112 colonnes ou 28 lignes, ou si le programme est déjà dans tmux, le système continue de lancer les labs selon l'ancien modèle. La variable LPIC_DAILY_LAB_LAYOUT=legacy force explicitement l'ancien modèle.

## Critères de recette à valider avant fusion

1. CI Go, formatage, validation des contenus et des tests.
2. Lab Podman réel : shell interactif, Ctrl+Z, nano/vi, redimensionnement, contrôle par :check, indices par :hint, nettoyage par :quit.
3. Lab 103.5 : tmux interne à la sandbox et Ctrl+B fonctionnent normalement.
4. Lab KVM : console série, reboot, redimensionnement et vérification compatibles avec l'affichage en deux panneaux.
5. Détacher/rattacher une session sans réinitialiser le lab ; fermer normalement et constater le nettoyage de la sandbox.
6. Sorties malformées : pas d'injection de codes de contrôle dans le panneau gauche.
7. Repli classique lorsque tmux manque ou lorsque la largeur n'est pas suffisante.

Cette branche est une proposition d'implémentation ; elle ne doit pas être fusionnée sans la recette interactive réelle.
