# LPIC Daily · Interface terminal

## Objectif

Mode **Operate** : l'utilisateur doit comprendre ce qu'il doit faire, agir dans le lab et identifier immédiatement ce qui bloque la validation. Cette interface privilégie les actions sur les explications secondaires.

## Hiérarchie et grammaire

- **Tableau de bord** : marque du produit et progression résumée ; liste compacte des activités ; détail d'une seule sélection ; action recommandée ; raccourcis accessibles. Aucun identifiant de contenu n'apparaît comme libellé.
- **Lab** : titre et contexte court ; MISSION ; RÉUSSITE avec critères observables ; raccourcis. Le terminal parent conserve la consigne ; le terminal enfant affiche seulement un rappel des commandes et le shell.
- **Validation** : nombre de critères réussis en tête, puis tous les points à corriger avant les points OK. Si une vérification comprend beaucoup de réussites, seules deux lignes OK sont détaillées et le reste est résumé. Les diagnostics techniques sont secondaires.
- **Succès** : état explicite et rappel pédagogique ; la progression reste enregistrée selon le flux métier existant.

## Codes visuels

| Sens | Signal texte | Couleur ANSI quand disponible |
|---|---|---|
| Navigation et section | MISSION, RÉUSSITE, curseur ▸ | Cyan gras |
| Action réussie | OK, bilan complet | Vert gras |
| Intervention requise | À CORRIGER, bilan incomplet | Rouge gras |
| Avertissement | Vérification interrompue | Jaune gras |
| Métadonnées et aide | Textes secondaires, détails techniques | Atténué |
| Informations principales | Titre / libellés | Gras ou couleur par défaut |

La couleur ne porte jamais seule le sens : le texte OK ou À CORRIGER est toujours présent. Aucune séquence ANSI dans une redirection, un pipe, TERM=dumb ou avec la variable NO_COLOR.

## Contraintes

- Toute information provenant des scénarios ou du sandbox est neutralisée avant affichage. Ne jamais interpréter ses séquences ANSI.
- L'UX ne change pas les critères déterministes, la sécurité des labs, le stockage des preuves, les commandes ni leur rôle.
- Fenêtres étroites : titres tronqués, élément sélectionné toujours visible ; en faible hauteur, masquer les détails secondaires sans cacher l'action.
- Pas d'animation pendant les checks. Afficher un état de vérification bref, puis un résultat stable.
- Garder les commandes clavier ↑↓ / j,k, Entrée, l, a, p, q et :check, :hint, :reset, :quit.

## Tests / vérification visuelle

Exécuter go test ./internal/tui ./cmd/lpic, vérifier un plan vide et chargé, les états de check tout-vert / partiellement échoué, les fenêtres 44x12 et 80x24, la présence de NO_COLOR et les sorties redirigées. Une exécution réelle avec backend Podman/libvirt est distincte des tests de rendu.
